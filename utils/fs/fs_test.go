/*
 * Copyright (c) 2022. Uniontech Software Ltd. All rights reserved.
 *
 * Author: Heysion Y. <heysion@deepin.com>
 *
 * Maintainer: Heysion Y. <heysion@deepin.com>
 *
 * SPDX-License-Identifier: GNU General Public License v3.0 or later
 */

package fs

import (
	"testing"
)

var testDataSet = []struct {
	in  string
	out string
}{
	{"/bin/bash.txt", "bash.txt"},
	{"/etc/fstab", "fstab"},
	{"/etc/systemd/system", "system"},
	{"/usr/lib/libc.so.1.1", "libc.so.1.1"},
}

//GetFileName
func TestGetFileName(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataSet {
		ret := GetFileName(tds.in)
		if ret != tds.out {
			t.Errorf("the key %v , ret %v", tds, ret)
		}
	}
}

var testDataSet2 = []struct {
	in  string
	out string
}{
	{"/bin/bash.txt", "/bin"},
	{"/etc/fstab", "/etc"},
	{"/etc/systemd/system", "/etc/systemd"},
	{"/usr/lib/libc.so.1.1", "/usr/lib"},
}

// GetFilePPath
func TestGetFilePPath(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataSet2 {
		ret := GetFilePPath(tds.in)
		if ret != tds.out {
			t.Errorf("the key %v , ret %v", tds, ret)
		}
	}
}
