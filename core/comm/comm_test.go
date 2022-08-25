/*
 * Copyright (c) 2022. Uniontech Software Ltd. All rights reserved.
 *
 * Author: Heysion Y. <heysion@deepin.com>
 *
 * Maintainer: Heysion Y. <heysion@deepin.com>
 *
 * SPDX-License-Identifier: GNU General Public License v3.0 or later
 */
package comm

import (
	"testing"
)

// ExecAndWait
func TestExecAndWait(t *testing.T) {
	if v1, v2, err := ExecAndWait(10, "ls", "-al"); err != nil {
		t.Error(err, v1, v2)
	}
}
