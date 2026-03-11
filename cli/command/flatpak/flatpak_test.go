/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package flatpak

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	if got := normalizeVersion("5.15-23.08"); got != "5.15.23.08" {
		t.Fatalf("unexpected normalized version: %s", got)
	}
	if got := normalizeVersion("1"); got != "1.0.0.0" {
		t.Fatalf("unexpected normalized version: %s", got)
	}
}

func TestParseMetadata(t *testing.T) {
	metadata := "[Application]\ncommand=app\nruntime=org.kde.Platform/x86_64/6.5\n"
	command, runtime, runtimeVersion, err := parseMetadata(metadata)
	if err != nil {
		t.Fatalf("parse metadata failed: %v", err)
	}
	if command != "app" || runtime != "org.kde.Platform" || runtimeVersion != "6.5" {
		t.Fatalf("unexpected metadata parse result: %q %q %q", command, runtime, runtimeVersion)
	}
}

func TestRunConvertInvalidAppID(t *testing.T) {
	opts := &convertOptions{version: "1.0.0.0"}
	if err := runConvert("../bad-app", opts); err == nil {
		t.Fatalf("expected invalid app id error, got nil")
	}
}

func TestConvertFlatpak(t *testing.T) {
	origRun := runCommand
	origRunDir := runCommandInDir
	origReadFile := readFile
	origWriteFile := writeFile
	origMkdirAll := mkdirAll
	origRemoveAll := removeAll
	defer func() {
		runCommand = origRun
		runCommandInDir = origRunDir
		readFile = origReadFile
		writeFile = origWriteFile
		mkdirAll = origMkdirAll
		removeAll = origRemoveAll
	}()

	tmpDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	runCommand = func(name string, args ...string) (string, string, error) {
		if name != "ostree" {
			return "", "", fmt.Errorf("unexpected command: %s", name)
		}
		joined := strings.Join(args, " ")
		if strings.Contains(joined, " pull ") {
			return "", "", nil
		}
		if strings.Contains(joined, " checkout ") {
			workDir := filepath.Join(tmpDir, "Org.Test.App")
			if err := os.MkdirAll(filepath.Join(workDir, "flatpak", "files", "share", "applications"), 0755); err != nil {
				return "", "", err
			}
			if err := os.WriteFile(filepath.Join(workDir, "flatpak", "metadata"), []byte("command=demo\nruntime=org.kde.Platform/x86_64/6.5\n"), 0644); err != nil {
				return "", "", err
			}
			desktop := "[Desktop Entry]\nExec=/app/bin/demo %U\n"
			if err := os.WriteFile(filepath.Join(workDir, "flatpak", "files", "share", "applications", "demo.desktop"), []byte(desktop), 0644); err != nil {
				return "", "", err
			}
			return "", "", nil
		}
		return "", "", nil
	}
	runCommandInDir = func(dir string, name string, args ...string) (string, string, error) {
		return "", "", nil
	}

	opts := &convertOptions{version: "1.2.3.4", build: false}
	if err := convertFlatpak("Org.Test.App", opts, filepath.Join(tmpDir, "cache"), "commit"); err != nil {
		t.Fatalf("convert flatpak failed: %v", err)
	}

	yamlPath := filepath.Join(tmpDir, "Org.Test.App", "linglong.yaml")
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read linglong yaml failed: %v", err)
	}
	yamlText := string(yamlData)
	if !strings.Contains(yamlText, "id: org.test.app") {
		t.Fatalf("expected lower-case app id in yaml, got: %s", yamlText)
	}
	if !strings.Contains(yamlText, "base: org.deepin.base.flatpak.kde/6.5.0") {
		t.Fatalf("unexpected base in yaml: %s", yamlText)
	}
	if !strings.Contains(yamlText, "command: [demo]") {
		t.Fatalf("unexpected command in yaml: %s", yamlText)
	}

	desktopPath := filepath.Join(tmpDir, "Org.Test.App", "flatpak", "files", "share", "applications", "demo.desktop")
	desktopData, err := os.ReadFile(desktopPath)
	if err != nil {
		t.Fatalf("read desktop failed: %v", err)
	}
	if !strings.Contains(string(desktopData), "Exec=/opt/apps/Org.Test.App/files/bin/demo") {
		t.Fatalf("desktop exec not rewritten: %s", string(desktopData))
	}
}
