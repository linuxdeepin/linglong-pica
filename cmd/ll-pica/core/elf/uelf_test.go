/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package elf

import (
	"testing"
)

var testDataSet = []struct {
	file  string
	isElf bool
}{
	{"/bin/bash", true},
	{"/etc/fstab", false},
}

var testDataSet2 = []struct {
	path  string
	count int
}{
	{"/usr/sbin", 5},
	{"/etc/", -1},
}

func TestIsElfWithPath(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataSet {
		ret := IsElfWithPath(tds.file)
		if ret != tds.isElf {
			t.Errorf("the key %v , ret %v", tds, ret)
		}
	}
}

func TestGetElfWithPath(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataSet2 {
		ret, err := GetElfWithPath(tds.path)
		if err != nil || len(ret) < tds.count {
			t.Errorf("the key %v , ret %v", tds, ret)
		}
	}
}
