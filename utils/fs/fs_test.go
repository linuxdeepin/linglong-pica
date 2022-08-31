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

// GetFileName
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
	{"/lib/x86_64-linux-gnu/", false},
}

// test CopyDirKeepPathAndPerm
func TestCopyDirKeepPathAndPerm(t *testing.T) {
	t.Parallel()
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

// test CopyDirKeepPathAndPerm
func TestCopyDirKeepPathAndPerm2(t *testing.T) {
	// t.Parallel()
	dst := "/tmp/aaaaaxxx"
	for _, tds := range testDataSet3 {
		fmt.Println(tds)
		if err := CopyDirKeepPathAndPerm("/mnt/workdir/rootfs/"+tds.in, dst, true, false, false); err != nil && tds.ret {
			t.Error("failed:", err, tds, dst)
		}
	}
	if err := os.RemoveAll(dst); err != nil {
		t.Error("failed:", err, dst)
	}

}

var testDataSet4 = []struct {
	in  string
	ret bool
}{
	{"/bin/bash.txt", false},
	{"/etc/fstab", true},
	{"/etc/systemd/system", false},
	{"/usr/lib/x86_64-linux-gnu/libc.so.6", true},
}

// test CopyFileKeepPermission
func TestCopyFileKeepPermission(t *testing.T) {
	dst := "/tmp/aaaaaxxx"

	for _, tds := range testDataSet4 {
		fmt.Println(tds)
		if err := CopyFileKeepPermission("/mnt/workdir/rootfs"+tds.in, dst, true, false); err != nil && tds.ret {
			t.Error("failed:", err, tds, dst)
		}
	}
}

// test IsDir
var testDataSet5 = []struct {
	in  string
	ret bool
}{
	{"/etc/default/grub.d", true},
	{"/bin/bash.txt", false},
	{"/etc/fstab", false},
	{"/usr/bin/", true},
}

func TestIsDir(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataSet5 {
		fmt.Println(tds)
		ret := IsDir(tds.in)
		if ret != tds.ret {
			t.Error("failed:", tds.in)
		}
	}
}

// test MoveFileOrDir
var testDataSet6 = []struct {
	src string
	dst string
	ret bool
}{
	{"/tmp/test-pica", "/tmp/test-pica1", true},
	{"/bin/bash.txt", "/tmp/bash.txt", false},
	{"/tmp/test-pica1/test-pica1.txt", "/tmp/test-pica/test-pica.txt", true},
}

func TestMoveFileOrDir(t *testing.T) {
	t.Parallel()
	// 测试目录移动
	if ret, err := CreateDir(testDataSet6[0].src); !ret && err != nil {
		t.Errorf("CreateDir failed! : %s", testDataSet6[0].src)
	}
	ret, err := MoveFileOrDir(testDataSet6[0].src, testDataSet6[0].dst)
	if ret != testDataSet6[0].ret && err != nil {
		t.Error("Test move dir failed! : ", testDataSet6[0].src)
	}

	// 测试移动文件
	f, err := os.Create(testDataSet6[2].src)
	if err != nil {
		t.Error("Create file failed! : ", testDataSet6[2].src)
	}
	defer f.Close()
	f.WriteString("test-pica")
	ret, err = MoveFileOrDir(testDataSet6[2].src, testDataSet6[2].dst)
	if ret != testDataSet6[2].ret && err != nil {
		t.Error("Test move file failed! : ", testDataSet6[2].src)
	}

	// 移动不存在文件
	ret, err = MoveFileOrDir(testDataSet6[1].src, testDataSet6[1].dst)
	if ret != testDataSet6[1].ret || err == nil {
		t.Error("Test move file failed! : ", testDataSet6[1].src)
	}

	// 移除创建的目录
	err = os.RemoveAll(testDataSet6[0].src)
	if err != nil {
		t.Error("remove dir failed ! : ", testDataSet6[0].src)
	}
	err = os.RemoveAll(testDataSet6[0].dst)
	if err != nil {
		t.Error("remove dir failed ! : ", testDataSet6[0].dst)
	}
}
