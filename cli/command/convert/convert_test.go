/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package convert

import (
	"strings"
	"testing"
)

func TestNewConvertCommandRemovesWithDepFlag(t *testing.T) {
	cmd := NewConvertCommand()
	if flag := cmd.Flags().Lookup("withDep"); flag != nil {
		t.Fatalf("expected withDep flag to be removed")
	}
}

func TestFormatRuntimeRef(t *testing.T) {
	if got := formatRuntimeRef("org.deepin.runtime.dtk", "25.2.1"); got != "org.deepin.runtime.dtk/25.2.1" {
		t.Fatalf("unexpected runtime ref: %q", got)
	}
	if got := formatRuntimeRef("org.deepin.runtime.dtk", ""); got != "" {
		t.Fatalf("expected empty runtime ref, got %q", got)
	}
}

func TestValidateDebArchitecture(t *testing.T) {
	if err := validateDebArchitecture("amd64", "amd64"); err != nil {
		t.Fatalf("expected matching arch to pass, got %v", err)
	}
	if err := validateDebArchitecture("", "amd64"); err != nil {
		t.Fatalf("expected empty deb arch to be ignored, got %v", err)
	}
	err := validateDebArchitecture("sw64", "amd64")
	if err == nil {
		t.Fatalf("expected mismatched arch to fail")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("unexpected mismatch error: %v", err)
	}
}
