/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package deb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDependencyNames(t *testing.T) {
	exclude := map[string]struct{}{
		"libc6": {},
	}

	got := parseDependencyNames([]string{
		"libc6 (>= 2.38), foo:any, pkg-a | pkg-b, debhelper (>= 13), debhelper-compat (= 13), bar <!nocheck>, baz [linux-any], ${misc:Depends}",
	}, exclude)

	want := []string{"foo", "pkg-a", "pkg-b", "bar", "baz"}
	if len(got) != len(want) {
		t.Fatalf("unexpected dependency count: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected dependency at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestCleanSourcePackage(t *testing.T) {
	tests := map[string]string{
		"bash (5.2.21-2.1)": "bash",
		"srcpkg:any":        "srcpkg",
		"":                  "",
	}

	for input, want := range tests {
		if got := cleanSourcePackage(input); got != want {
			t.Fatalf("cleanSourcePackage(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGenerateBuildScriptDropsLegacyDependencyInstall(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	sourceDir := filepath.Join(root, "sources", "demo", "usr", "share", "applications")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	desktopPath := filepath.Join(sourceDir, "demo.desktop")
	desktop := `[Desktop Entry]
Type=Application
Name=Demo
Exec=/usr/bin/demo --flag
Icon=/usr/share/icons/demo
`
	if err := os.WriteFile(desktopPath, []byte(desktop), 0o644); err != nil {
		t.Fatalf("write desktop failed: %v", err)
	}

	d := &Deb{
		Name: "demo",
		Id:   "io.demo.app",
		Path: filepath.Join(root, "sources", "demo_1.0.0_amd64.deb"),
	}

	d.GenerateBuildScript()

	buildScript := strings.Join(d.Build, "\n")
	for _, forbidden := range []string{"DEPS_LIST", "find $SOURCES", "ar -x", "patchelf --set-rpath"} {
		if strings.Contains(buildScript, forbidden) {
			t.Fatalf("build script still contains legacy dependency install fragment %q:\n%s", forbidden, buildScript)
		}
	}

	if !strings.Contains(buildScript, "cp -r $EXTERNAL_DEB_SOURCES/demo/usr/* $PREFIX") {
		t.Fatalf("expected build script to copy unpacked usr content, got:\n%s", buildScript)
	}
	if len(d.Command) == 0 || d.Command[0] != "/opt/apps/io.demo.app/files/bin/demo" {
		t.Fatalf("unexpected generated command: %v", d.Command)
	}
}

func TestNormalizeExecLineForUsrBinary(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "usr", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	binPath := filepath.Join(binDir, "iptux")
	if err := os.WriteFile(binPath, []byte{}, 0o755); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	got := normalizeExecLine("iptux --minimized", root, "io.github.iptux")
	want := "/opt/apps/io.github.iptux/files/bin/iptux --minimized"
	if got != want {
		t.Fatalf("normalizeExecLine() = %q, want %q", got, want)
	}
}

func TestNormalizeExecLineDoesNotRewriteUsrLibOrShare(t *testing.T) {
	root := t.TempDir()
	libDir := filepath.Join(root, "usr", "lib")
	shareDir := filepath.Join(root, "usr", "share")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("mkdir lib failed: %v", err)
	}
	if err := os.MkdirAll(shareDir, 0o755); err != nil {
		t.Fatalf("mkdir share failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "helper"), []byte{}, 0o755); err != nil {
		t.Fatalf("write lib helper failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(shareDir, "demo"), []byte{}, 0o755); err != nil {
		t.Fatalf("write share demo failed: %v", err)
	}

	for _, execLine := range []string{"helper --flag", "demo --flag"} {
		if got := normalizeExecLine(execLine, root, "io.demo.app"); got != execLine {
			t.Fatalf("normalizeExecLine(%q) = %q, want unchanged", execLine, got)
		}
	}
}

func TestNormalizeExecLineKeepsOptAppsExecPath(t *testing.T) {
	root := t.TempDir()
	execLine := "/opt/apps/old.app/files/bin/demo --flag"

	got := normalizeExecLine(execLine, root, "io.demo.app")
	if got != execLine {
		t.Fatalf("normalizeExecLine(%q) = %q, want unchanged", execLine, got)
	}
}

func TestCheckDebHashRejectsEmptyFile(t *testing.T) {
	root := t.TempDir()
	debPath := filepath.Join(root, "demo.deb")
	if err := os.WriteFile(debPath, nil, 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	d := &Deb{
		Name: "demo",
		Path: debPath,
	}
	if d.CheckDebHash() {
		t.Fatalf("expected empty deb file to be rejected")
	}
}
