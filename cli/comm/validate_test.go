/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package comm

import "testing"

func TestValidatePackageID(t *testing.T) {
	valid := []string{"io.demo.app", "Org_Test-App", "a", "A1_b.c-d"}
	for _, id := range valid {
		if err := ValidatePackageID(id); err != nil {
			t.Fatalf("expected valid id %q, got error: %v", id, err)
		}
	}

	invalid := []string{"", "../tmp", "/tmp/app", "a/b", "a b", "~app", ".hidden", "中文"}
	for _, id := range invalid {
		if err := ValidatePackageID(id); err == nil {
			t.Fatalf("expected invalid id %q to fail", id)
		}
	}
}
