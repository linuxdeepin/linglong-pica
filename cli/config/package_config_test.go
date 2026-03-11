/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageConfigKeepsEmptyRuntimeVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package.yaml")

	pack := NewPackConfig()
	pack.Runtime.Config.RuntimeVersion = ""
	if !pack.CreatePackConfigYaml(path) {
		t.Fatalf("CreatePackConfigYaml returned false")
	}

	loaded := NewPackConfig()
	loaded.Runtime.Config.RuntimeVersion = "25.2.1"
	if !loaded.ReadPackConfigYaml(path) {
		t.Fatalf("ReadPackConfigYaml returned false")
	}
	if loaded.Runtime.Config.RuntimeVersion != "" {
		t.Fatalf("expected empty runtime version, got %q", loaded.Runtime.Config.RuntimeVersion)
	}
}

func TestPackageConfigOmitsLegacyRepoFieldsWhenWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package.yaml")

	pack := NewPackConfig()
	if !pack.CreatePackConfigYaml(path) {
		t.Fatalf("CreatePackConfigYaml returned false")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read package config failed: %v", err)
	}
	content := string(data)
	for _, forbidden := range []string{"source:", "distro_version:", "\n  version:"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("expected generated package config to omit %q, got:\n%s", forbidden, content)
		}
	}
	if !strings.Contains(content, "runtime_version: \"\"") {
		t.Fatalf("expected generated package config to use runtime_version, got:\n%s", content)
	}
}

func TestPackageConfigReadsLegacyRepoFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package.yaml")
	legacy := `runtime:
  version: "25.2.1"
  base_version: 25.2.1
  source: https://legacy.example/repo
  distro_version: crimson/appstore
  arch: amd64
file:
  deb:
    - type: repo
      id: com.example.app
      name: com.example.app
`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy config failed: %v", err)
	}

	loaded := NewPackConfig()
	if !loaded.ReadPackConfigYaml(path) {
		t.Fatalf("ReadPackConfigYaml returned false")
	}
	if loaded.Runtime.Config.Source != "https://legacy.example/repo" {
		t.Fatalf("unexpected legacy source: %q", loaded.Runtime.Config.Source)
	}
	if loaded.Runtime.Config.DistroVersion != "crimson/appstore" {
		t.Fatalf("unexpected legacy distro_version: %q", loaded.Runtime.Config.DistroVersion)
	}
	if loaded.Runtime.Config.RuntimeVersion != "25.2.1" {
		t.Fatalf("unexpected legacy runtime version: %q", loaded.Runtime.Config.RuntimeVersion)
	}
}
