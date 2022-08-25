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
	"fmt"
	"os"
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

var testDataSet3 = []struct {
	in  string
	ret bool
}{
	{"/etc/default/grub.d", true},
	{"/bin/bash.txt", false},
	{"/etc/fstab", false},
	{"/usr/bin/", true},
}

// test CopyDirKeepPathAndPerm
func TestCopyDirKeepPathAndPerm(t *testing.T) {
	// t.Parallel()
	dst := "/tmp/aaaaaxxx"
	for _, tds := range testDataSet3 {
		fmt.Println(tds)
		if err := CopyDirKeepPathAndPerm(tds.in, dst, true, false, false); err != nil && tds.ret {
			t.Error("failed:", err, tds, dst)
		}
	}
	if err := os.RemoveAll(dst); err != nil {
		t.Error("failed:", err, dst)
	}

}
