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
	"io/ioutil"
	"os"
	"testing"
)

// test IsDir
var testDataIsDir = []struct {
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
	for _, tds := range testDataIsDir {
		fmt.Println(tds)
		ret := IsDir(tds.in)
		if ret != tds.ret {
			t.Error("failed:", tds.in)
		}
	}
}

// CheckFileExits
var testDataCheckFileExits = []struct {
	in  string
	out bool
}{
	{"/bin/bash.txt", false},
	{"/etc/fstab", true},
	{"/etc/systemd/system", true},
	{"/tmp/ll-pica", false},
}

func TestCheckFileExits(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataCheckFileExits {
		if ret, err := CheckFileExits(tds.in); err != nil && tds.out || ret != tds.out {
			t.Errorf("Failed test for CheckFileExits!")
		}
	}

}

// CreateDir
var testDataCreateDir = []struct {
	in  string
	out bool
}{
	{"/tmp/ll-pica", true},
	{"/tmp/ll-lingong", true},
	{"/etc/apt/sources.list", false},
}

func TestCreateDir(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataCreateDir {
		if ret, err := CreateDir(tds.in); err != nil && tds.out || ret != tds.out {
			t.Errorf("Failed test for CreateDir! Error: %+v", tds.in)
		} else if ret {
			if ret, err := RemovePath(tds.in); !ret && err != nil {
				t.Errorf("Failed test for CreateDir! Error: failed to remove %+v", tds.in)
			}
		}
	}

}

// RemovePath
func TestRemovePath(t *testing.T) {
	t.Parallel()
	// 目录测试
	testDirPath := "/tmp/ll-pica"
	if ret, err := CreateDir(testDirPath); err != nil && !ret {
		t.Errorf("Failed test for RemovePath! Error: create dir err of %+v", testDirPath)
	}
	if ret, err := RemovePath(testDirPath); !ret && err != nil {
		t.Errorf("Failed test for RemovePath! Error: failed to remove %+v", testDirPath)
	}
	// 测试文件
	testFilePath := "/tmp/ll-pica.txt"
	if err := ioutil.WriteFile(testFilePath, []byte("I am testing!"), 0644); err != nil {
		t.Errorf("Failed test for RemovePath! Error: failed to write file of  %+v", testFilePath)
	}
	if ret, err := RemovePath(testFilePath); !ret && err != nil {
		t.Errorf("Failed test for RemovePath! Error: failed to remove %+v", testFilePath)
	}
	// 测试不存在文件
	testFilePath = "/tmp/ll-linglong.txt"
	if ret, err := RemovePath(testFilePath); ret || err == nil {
		t.Errorf("Failed test for RemovePath! Error: failed to remove %+v", testFilePath)
	}
}

// GetFileName
var testDataGetFileName = []struct {
	in  string
	out string
}{
	{"/bin/bash.txt", "bash.txt"},
	{"/etc/fstab", "fstab"},
	{"/etc/systemd/system", "system"},
	{"/usr/lib/libc.so.1.1", "libc.so.1.1"},
}

func TestGetFileName(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataGetFileName {
		ret := GetFileName(tds.in)
		if ret != tds.out {
			t.Errorf("the key %v , ret %v", tds, ret)
		}
	}
}

// GetFilePPath
var testDataGetFilePPath = []struct {
	in  string
	out string
}{
	{"/bin/bash.txt", "/bin"},
	{"/etc/fstab", "/etc"},
	{"/etc/systemd/system", "/etc/systemd"},
	{"/usr/lib/libc.so.1.1", "/usr/lib"},
}

func TestGetFilePPath(t *testing.T) {
	t.Parallel()
	for _, tds := range testDataGetFilePPath {
		ret := GetFilePPath(tds.in)
		if ret != tds.out {
			t.Errorf("the key %v , ret %v", tds, ret)
		}
	}
}

// test MoveFileOrDir
var testDataMoveFileOrDir = []struct {
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
	if ret, err := CreateDir(testDataMoveFileOrDir[0].src); !ret && err != nil {
		t.Errorf("CreateDir failed! : %s", testDataMoveFileOrDir[0].src)
	}
	ret, err := MoveFileOrDir(testDataMoveFileOrDir[0].src, testDataMoveFileOrDir[0].dst)
	if ret != testDataMoveFileOrDir[0].ret && err != nil {
		t.Error("Test move dir failed! : ", testDataMoveFileOrDir[0].src)
	}

	// 测试移动文件
	f, err := os.Create(testDataMoveFileOrDir[2].src)
	if err != nil {
		t.Error("Create file failed! : ", testDataMoveFileOrDir[2].src)
	}
	defer f.Close()
	f.WriteString("test-pica")
	ret, err = MoveFileOrDir(testDataMoveFileOrDir[2].src, testDataMoveFileOrDir[2].dst)
	if ret != testDataMoveFileOrDir[2].ret && err != nil {
		t.Error("Test move file failed! : ", testDataMoveFileOrDir[2].src)
	}

	// 移动不存在文件
	ret, err = MoveFileOrDir(testDataMoveFileOrDir[1].src, testDataMoveFileOrDir[1].dst)
	if ret != testDataMoveFileOrDir[1].ret || err == nil {
		t.Error("Test move file failed! : ", testDataMoveFileOrDir[1].src)
	}

	// 移除创建的目录
	err = os.RemoveAll(testDataMoveFileOrDir[0].src)
	if err != nil {
		t.Error("remove dir failed ! : ", testDataMoveFileOrDir[0].src)
	}
	err = os.RemoveAll(testDataMoveFileOrDir[0].dst)
	if err != nil {
		t.Error("remove dir failed ! : ", testDataMoveFileOrDir[0].dst)
	}
}

// CopyFile
var testDataCopyFile = []struct {
	in  string
	out string
	ret bool
}{
	{"/tmp/ll-pica.txt", "/tmp/ll-pica1.txt", true},
	{"/tmp/ll-lingong.txt", "/tmp/ll-lingong1.txt", false},
}

func TestCopyFile(t *testing.T) {
	t.Parallel()
	// 测试已存在文件拷贝
	if err := ioutil.WriteFile(testDataCopyFile[0].in, []byte("ll-pica testing"), 0644); err != nil {
		t.Errorf("Failed test for TestCopyFile! Error: failed to write %+v", testDataCopyFile[0].in)
	}
	if ret, err := CopyFile(testDataCopyFile[0].in, testDataCopyFile[0].out); err != nil || !ret || ret != testDataCopyFile[0].ret {
		t.Errorf("Failed test for TestCopyFile! Error: failed to CopyFile %+v", testDataCopyFile[0].in)
	}
	// 判断文件权限
	srcFile, err := os.Open(testDataCopyFile[0].in)
	if err != nil {
		t.Errorf("Failed test for TestCopyFile! Error: failed to open %+v", testDataCopyFile[0].in)
	}
	defer srcFile.Close()
	fi1, _ := srcFile.Stat()
	perm1 := fi1.Mode()

	dstFile, err := os.Open(testDataCopyFile[0].out)
	if err != nil {
		t.Errorf("Failed test for TestCopyFile! Error: failed to open %+v", testDataCopyFile[0].out)
	}
	defer dstFile.Close()
	fi2, _ := dstFile.Stat()
	perm2 := fi2.Mode()
	if perm1 != perm2 {
		t.Errorf("Failed test for TestCopyFile! Error: failed to copy perm %+v", testDataCopyFile[0].out)
	}

	// 移除产生的文件
	if ret, err := RemovePath(testDataCopyFile[0].in); !ret || err != nil {
		t.Errorf("Failed test for TestCopyFile! Error: failed to remove %+v", testDataCopyFile[0].in)
	}
	if ret, err := RemovePath(testDataCopyFile[0].out); !ret || err != nil {
		t.Errorf("Failed test for TestCopyFile! Error: failed to remove %+v", testDataCopyFile[0].out)
	}

	// 测试不存在的文件
	if ret, err := CopyFile(testDataCopyFile[1].in, testDataCopyFile[1].out); err == nil || ret || ret != testDataCopyFile[1].ret {
		t.Errorf("Failed test for TestCopyFile! Error: failed to CopyFile %+v", testDataCopyFile[1].in)
	}
}

// CopyDir
var testDataCopyDir = []struct {
	in  string
	out string
	ret bool
}{
	{"/tmp/ll-pica", "/tmp/ll-pica1", true},
	{"/tmp/ll-linglong-test", "/tmp/ll-linglong-test1", false},
}

func TestCopyDir(t *testing.T) {
	t.Parallel()
	// 测试已存在的目录
	if ret, err := CreateDir(testDataCopyDir[0].in); err != nil || !ret {
		t.Errorf("Failed test for TestCopyDir! Error: failed to create dir %+v", testDataCopyDir[0].in)
	}
	if ret := CopyDir(testDataCopyDir[0].in, testDataCopyDir[0].out); !ret || ret != testDataCopyDir[0].ret {
		t.Errorf("Failed test for TestCopyDir! Error: failed to copy dir %+v", testDataCopyDir[0].in)
	}
	// 移除产生的目录
	if ret, err := RemovePath(testDataCopyDir[0].in); !ret || err != nil {
		t.Errorf("Failed test for TestCopyDir! Error: failed to remove %+v", testDataCopyDir[0].in)
	}
	if ret, err := RemovePath(testDataCopyDir[0].out); !ret || err != nil {
		t.Errorf("Failed test for TestCopyDir! Error: failed to remove %+v", testDataCopyDir[0].out)
	}

	//测试不存在的目录
	if ret := CopyDir(testDataCopyDir[1].in, testDataCopyDir[1].out); ret || ret != testDataCopyDir[1].ret {
		t.Errorf("Failed test for TestCopyDir! Error: failed to copy dir %+v", testDataCopyDir[1].in)
	}
}

// test CopyFileKeepPermission
var testDataCopyFileKeepPermission = []struct {
	in  string
	ret bool
}{
	{"/bin/bash.txt", false},
	{"/etc/fstab", true},
	{"/etc/systemd/system", false},
	{"/usr/lib/x86_64-linux-gnu/libc.so.6", true},
}

func TestCopyFileKeepPermission(t *testing.T) {
	dst := "/tmp/aaaaaxxx"

	for _, tds := range testDataCopyFileKeepPermission {
		fmt.Println(tds)
		if err := CopyFileKeepPermission("/mnt/workdir/rootfs"+tds.in, dst, true, false); err != nil && tds.ret {
			t.Error("failed:", err, tds, dst)
		}
	}
}

// CopyDirKeepPathAndPerm
var testDataCopyDirKeepPathAndPerm = []struct {
	in  string
	ret bool
}{
	{"/etc/default/grub.d", true},
	{"/bin/bash.txt", false},
	{"/etc/fstab", false},
	{"/usr/bin/", true},
	{"/lib/x86_64-linux-gnu/", false},
}

func TestCopyDirKeepPathAndPerm(t *testing.T) {
	t.Parallel()
	dst := "/tmp/aaaaaxxx"
	for _, tds := range testDataCopyDirKeepPathAndPerm {
		fmt.Println(tds)
		if err := CopyDirKeepPathAndPerm(tds.in, dst, true, false, false); err != nil && tds.ret {
			t.Error("failed:", err, tds, dst)
		}
	}
	if err := os.RemoveAll(dst); err != nil {
		t.Error("failed:", err, dst)
	}

}

func TestCopyDirKeepPathAndPerm2(t *testing.T) {
	// t.Parallel()
	dst := "/tmp/aaaaaxxx"
	for _, tds := range testDataCopyDirKeepPathAndPerm {
		fmt.Println(tds)
		if err := CopyDirKeepPathAndPerm("/mnt/workdir/rootfs/"+tds.in, dst, true, false, false); err != nil && tds.ret {
			t.Error("failed:", err, tds, dst)
		}
	}
	if err := os.RemoveAll(dst); err != nil {
		t.Error("failed:", err, dst)
	}

}

// FindBundlePath
// HasBundleName
// DesktopInit
// DesktopGroupname
// TransExecToLl
// TransIconToLl
