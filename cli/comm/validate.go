/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package comm

import (
	"fmt"
	"regexp"
)

var packageIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// ValidatePackageID validates user-provided package identifiers before using
// them in paths, shell commands, or generated build metadata.
func ValidatePackageID(id string) error {
	if id == "" {
		return fmt.Errorf("package id is required")
	}

	if !packageIDPattern.MatchString(id) {
		return fmt.Errorf("invalid package id: %s (must start with alphanumeric character and contain only alphanumeric, dot, underscore or hyphen, max 128 characters)", id)
	}
	return nil
}
