/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package comm

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigUnmarshalYAMLReadsLegacyVersion(t *testing.T) {
	var cfg Config
	data := []byte(`runtime:
  version: "25.2.1"
  base_version: 25.2.1
  arch: amd64
`)

	var wrapper struct {
		Runtime Config `yaml:"runtime"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		t.Fatalf("yaml unmarshal failed: %v", err)
	}

	cfg = wrapper.Runtime
	if cfg.RuntimeVersion != "25.2.1" {
		t.Fatalf("expected legacy version to map to runtime version, got %q", cfg.RuntimeVersion)
	}
}

func TestConfigJSONUsesRuntimeVersionAndReadsLegacyVersion(t *testing.T) {
	cfg := Config{
		RuntimeVersion: "25.2.1",
		BaseVersion:    "25.2.1",
		Arch:           "amd64",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}
	if string(data) == "" {
		t.Fatal("expected marshaled json to be non-empty")
	}
	if contains := string(data); contains == "" {
		t.Fatal("expected marshaled json to contain runtime_version")
	}
	if !jsonContainsKey(data, "runtime_version") {
		t.Fatalf("expected json to contain runtime_version, got %s", data)
	}
	if jsonContainsKey(data, "version") {
		t.Fatalf("expected json to omit legacy version key, got %s", data)
	}

	var legacy Config
	if err := json.Unmarshal([]byte(`{"version":"23.0.0","base_version":"25.2.1","arch":"amd64"}`), &legacy); err != nil {
		t.Fatalf("json unmarshal legacy config failed: %v", err)
	}
	if legacy.RuntimeVersion != "23.0.0" {
		t.Fatalf("expected legacy version to map to runtime version, got %q", legacy.RuntimeVersion)
	}
}

func jsonContainsKey(data []byte, key string) bool {
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return false
	}
	_, ok := decoded[key]
	return ok
}
