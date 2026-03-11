/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package flatpak

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/tools/log"
)

const flatpakConfigName = "ll-pica-flatpak-config.json"

var (
	runCommand      = defaultRunCommand
	runCommandInDir = defaultRunCommandInDir
	removeAll       = os.RemoveAll
	mkdirAll        = os.MkdirAll
	readFile        = os.ReadFile
	writeFile       = os.WriteFile
)

type convertOptions struct {
	base        string
	baseVersion string
	version     string
	build       bool
	layer       bool
}

type flatpakConfig struct {
	Flathub struct {
		URL string `json:"url"`
	} `json:"flathub"`
}

func NewFlatpakCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "flatpak",
		Short: "Convert flatpak package to linglong package",
	}

	cmd.AddCommand(newConvertCommand())
	return cmd
}

func newConvertCommand() *cobra.Command {
	var options convertOptions
	cmd := &cobra.Command{
		Use:          "convert [APPID]",
		Short:        "Convert flatpak to uab",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConvert(args[0], &options)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&options.base, "base", "", "override linglong base name")
	flags.StringVar(&options.baseVersion, "base-version", "", "override linglong base version")
	flags.StringVar(&options.version, "version", "1.0.0.0", "linglong package version")
	flags.BoolVar(&options.build, "build", false, "build linglong")
	flags.BoolVar(&options.layer, "layer", false, "export layer file")
	return cmd
}

func runConvert(appID string, options *convertOptions) error {
	if err := comm.ValidatePackageID(appID); err != nil {
		return err
	}
	log.Logger.Infof("starting conversion for package %s", appID)

	if options.version == "" {
		options.version = "1.0.0.0"
	}
	options.version = normalizeVersion(options.version)

	configPath, flathubURL, err := ensureFlatpakConfig()
	if err != nil {
		return err
	}
	_ = configPath

	flathubCache := filepath.Join(os.Getenv("HOME"), ".cache", "linglong-pica-flathub")
	if err := mkdirAll(flathubCache, 0755); err != nil {
		return err
	}

	if _, stderr, err := runCommand("ostree", "init", "--repo="+flathubCache, "--mode", "bare-user-only"); err != nil {
		return fmt.Errorf("ostree init failed: %s: %w", stderr, err)
	}

	if _, _, err := runCommand("ostree", "--repo="+flathubCache, "remote", "delete", "--if-exists", "flathub"); err != nil {
		return err
	}
	if _, stderr, err := runCommand("ostree", "--repo="+flathubCache, "remote", "add", "--no-sign-verify", "flathub", flathubURL); err != nil {
		return fmt.Errorf("ostree remote add failed: %s: %w", stderr, err)
	}

	_, _, _ = runCommand("flatpak", "--user", "remote-delete", "ll-pica")
	defer runCommand("flatpak", "--user", "remote-delete", "ll-pica")

	if _, stderr, err := runCommand("flatpak", "--user", "remote-add", "--no-gpg-verify", "ll-pica", flathubURL); err != nil {
		return fmt.Errorf("flatpak remote add failed: %s: %w", stderr, err)
	}

	commitID, stderr, err := runCommand("flatpak", "--user", "remote-info", "ll-pica", appID, "--show-commit")
	if err != nil {
		return fmt.Errorf("fetch commit id failed: %s: %w", stderr, err)
	}
	commitID = strings.TrimSpace(commitID)
	if commitID == "" {
		return fmt.Errorf("fetch commit id failed: empty commit id")
	}

	return convertFlatpak(appID, options, flathubCache, commitID)
}

func convertFlatpak(appID string, options *convertOptions, flathubCache string, commitID string) error {
	if _, stderr, err := runCommand("ostree", "--repo="+flathubCache, "pull", "flathub", commitID); err != nil {
		return fmt.Errorf("ostree pull failed: %s: %w", stderr, err)
	}

	workDir := appID
	if err := removeAll(workDir); err != nil {
		return err
	}
	if err := mkdirAll(workDir, 0755); err != nil {
		return err
	}

	flatpakDir := filepath.Join(workDir, "flatpak")
	if _, stderr, err := runCommand("ostree", "--repo="+flathubCache, "checkout", commitID, flatpakDir); err != nil {
		return fmt.Errorf("ostree checkout failed: %s: %w", stderr, err)
	}

	metadataPath := filepath.Join(flatpakDir, "metadata")
	metadataData, err := readFile(metadataPath)
	if err != nil {
		return err
	}
	flatpakCommand, flatpakRuntime, flatpakRuntimeVersion, err := parseMetadata(string(metadataData))
	if err != nil {
		return err
	}

	linglongBaseName := inferBaseName(flatpakRuntime)
	linglongBaseVersion := inferBaseVersion(flatpakRuntimeVersion)
	if options.base != "" {
		linglongBaseName = options.base
	}
	if options.baseVersion != "" {
		linglongBaseVersion = options.baseVersion
	}

	execOld, err := rewriteDesktopExec(workDir, appID)
	if err != nil {
		return err
	}

	binFilePath := inferBinFilePath(execOld, appID)
	smallerAppID := strings.ToLower(appID)
	log.Logger.Infof("%s: generating linglong.yaml", appID)
	linglongYaml := fmt.Sprintf(`version: "1"
package:
  id: %s
  name: %s
  version: %s
  kind: app
  description: flatpak runtime environment on linglong

command: [%s]
base: %s/%s

build: |
  mkdir $PREFIX/etc
  cp profile $PREFIX/etc
  cp -rf flatpak/files/* $PREFIX
`, smallerAppID, smallerAppID, options.version, flatpakCommand, linglongBaseName, linglongBaseVersion)

	profile := fmt.Sprintf(`#!/bin/sh
# bind /opt/apps/%s/files to /app
ln -s "/opt/apps/$LINGLONG_APPID/files" /run/linglong/app
`, appID)

	if err := writeFile(filepath.Join(workDir, "linglong.yaml"), []byte(linglongYaml), 0644); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(workDir, "profile"), []byte(profile), 0644); err != nil {
		return err
	}
	log.Logger.Infof("%s: generated linglong.yaml", appID)

	_ = binFilePath // keep inferred path behavior aligned with historical script side effects.

	if options.build {
		log.Logger.Infof("%s: building package", appID)
		if _, stderr, err := runCommandInDir(workDir, "ll-builder", "build"); err != nil {
			return fmt.Errorf("ll-builder build failed: %s: %w", stderr, err)
		}
		exportArgs := []string{"export"}
		if options.layer {
			exportArgs = append(exportArgs, "--layer")
		}
		log.Logger.Infof("%s: exporting package (%s)", appID, map[bool]string{true: "layer", false: "uab"}[options.layer])
		if _, stderr, err := runCommandInDir(workDir, "ll-builder", exportArgs...); err != nil {
			return fmt.Errorf("ll-builder export failed: %s: %w", stderr, err)
		}
		log.Logger.Infof("%s: export completed", appID)
	} else {
		log.Logger.Infof("%s: skip build/export (set --build to enable)", appID)
	}

	return nil
}

func ensureFlatpakConfig() (string, string, error) {
	if err := mkdirAll(comm.PicaConfigPath(), 0755); err != nil {
		return "", "", err
	}

	configPath := filepath.Join(comm.PicaConfigPath(), flatpakConfigName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := flatpakConfig{}
		defaultConfig.Flathub.URL = "https://dl.flathub.org/repo/"
		data, err := json.MarshalIndent(defaultConfig, "", "    ")
		if err != nil {
			return "", "", err
		}
		if err := writeFile(configPath, append(data, '\n'), 0644); err != nil {
			return "", "", err
		}
	}

	data, err := readFile(configPath)
	if err != nil {
		return "", "", err
	}

	cfg := flatpakConfig{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", "", err
	}
	if cfg.Flathub.URL == "" {
		cfg.Flathub.URL = "https://dl.flathub.org/repo/"
	}
	return configPath, cfg.Flathub.URL, nil
}

func normalizeVersion(version string) string {
	re := regexp.MustCompile(`[0-9]+`)
	digits := re.FindAllString(version, -1)
	parts := []string{"0", "0", "0", "0"}
	for i := 0; i < len(parts) && i < len(digits); i++ {
		parts[i] = digits[i]
	}
	return strings.Join(parts, ".")
}

func parseMetadata(metadata string) (command string, runtime string, runtimeVersion string, err error) {
	for _, line := range strings.Split(metadata, "\n") {
		if strings.HasPrefix(line, "command=") {
			command = strings.TrimSpace(strings.TrimPrefix(line, "command="))
		}
		if strings.HasPrefix(line, "runtime=") {
			rv := strings.TrimSpace(strings.TrimPrefix(line, "runtime="))
			parts := strings.Split(rv, "/")
			if len(parts) >= 3 {
				runtime = parts[0]
				runtimeVersion = parts[2]
			}
		}
	}

	if command == "" {
		return "", "", "", fmt.Errorf("flatpak metadata command is missing")
	}
	if runtime == "" || runtimeVersion == "" {
		return "", "", "", fmt.Errorf("flatpak metadata runtime is missing")
	}
	return command, runtime, runtimeVersion, nil
}

func inferBaseName(runtime string) string {
	parts := strings.Split(runtime, ".")
	segment := "runtime"
	if len(parts) > 1 && parts[1] != "" {
		segment = parts[1]
	}
	return "org.deepin.base.flatpak." + segment
}

func inferBaseVersion(runtimeVersion string) string {
	parts := regexp.MustCompile(`[-.]`).Split(runtimeVersion, -1)
	result := []string{"0", "0", "0"}
	idx := 0
	for _, part := range parts {
		if part == "" {
			continue
		}
		result[idx] = part
		idx++
		if idx == 3 {
			break
		}
	}
	return strings.Join(result, ".")
}

func rewriteDesktopExec(workDir string, appID string) (string, error) {
	desktops, err := filepath.Glob(filepath.Join(workDir, "flatpak", "files", "share", "applications", "*.desktop"))
	if err != nil {
		return "", err
	}
	if len(desktops) == 0 {
		return "", fmt.Errorf("desktop file not found")
	}

	lastExecOld := ""
	for _, desktop := range desktops {
		data, err := readFile(desktop)
		if err != nil {
			return "", err
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if !strings.HasPrefix(line, "Exec=") {
				continue
			}
			execOld := stripDesktopExecFieldCode(strings.TrimPrefix(line, "Exec="))
			lastExecOld = execOld
			execNew := execOld
			if strings.Contains(execOld, "/app") {
				execNew = strings.ReplaceAll(execOld, "/app", "/opt/apps/"+appID+"/files")
			}
			lines[i] = "Exec=" + execNew
			break
		}
		content := strings.Join(lines, "\n")
		if err := writeFile(desktop, []byte(content), 0644); err != nil {
			return "", err
		}
	}

	if lastExecOld == "" {
		return "", fmt.Errorf("desktop exec is missing")
	}
	return lastExecOld, nil
}

func stripDesktopExecFieldCode(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '%' && i+1 < len(value) {
			i++
			continue
		}
		b.WriteByte(value[i])
	}
	return strings.TrimSpace(b.String())
}

func inferBinFilePath(execOld string, appID string) string {
	fields := strings.Fields(execOld)
	if len(fields) == 0 {
		return filepath.Join(appID, "flatpak", "files", "bin")
	}
	binFile := fields[0]
	if strings.HasPrefix(binFile, "/") {
		return strings.ReplaceAll(binFile, "/app", appID+"/flatpak/files")
	}
	return filepath.Join(appID, "flatpak", "files", "bin", binFile)
}

func defaultRunCommand(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func defaultRunCommandInDir(dir string, name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}
