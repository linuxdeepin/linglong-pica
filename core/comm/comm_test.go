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
	"io/ioutil"
	. "ll-pica/utils/fs"
	"log"
	"os"
	"testing"
)

// ExecAndWait
func TestExecAndWait(t *testing.T) {
	if v1, v2, err := ExecAndWait(10, "ls", "-al"); err != nil {
		t.Error(err, v1, v2)
	}
}

// GetFileSha256
func TestGetFileSha256(t *testing.T) {
	oneFileSha256 := "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
	file, err := ioutil.TempFile("/tmp/", "sha256_")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	file.WriteString("1")

	file.Close()

	if fileSha256, err := GetFileSha256(file.Name()); fileSha256 != oneFileSha256 || err != nil {
		t.Error("failed: ", err, file.Name())
	}
}

// WriteRootfsRepo
func TestWriteRootfsRepo(t *testing.T) {
	sourceRepo := "deb [trusted=yes] http://pools.uniontech.com/desktop-professional/ eagle main contrib non-free"
	rootfsPath := "/tmp/ll-rootfs"
	var configTest Config
	var extraInfo ExtraInfo
	configTest.Rootfsdir = rootfsPath
	extraInfo.Repo = append(extraInfo.Repo, sourceRepo)
	if ret := extraInfo.WriteRootfsRepo(configTest); ret {
		t.Errorf("failed test for WriteRootfsRepo!")
	}
	if ret, err := CreateDir("/tmp/ll-rootfs/etc/apt"); !ret && err != nil {
		t.Errorf("failed test for WriteRootfsRepo! Error: failed to create dir")
	}
	if file, err := os.OpenFile(rootfsPath+"/etc/apt/sources.list", os.O_RDWR|os.O_APPEND|os.O_TRUNC|os.O_CREATE, 0644); err != nil {
		t.Errorf("failed test for WriteRootfsRepo! Error: failed to open sources.list")
	} else {
		defer file.Close()
		if _, err := file.WriteString("test ll-pica"); err != nil {
			t.Errorf("failed test for WriteRootfsRepo! Error: failed to write sources.list")
		}
		file.Sync()
		if ret := extraInfo.WriteRootfsRepo(configTest); !ret {
			t.Errorf("failed test for WriteRootfsRepo!")
		}
	}

	sourcesData, err := ioutil.ReadFile(rootfsPath + "/etc/apt/sources.list")
	if err != nil {
		t.Errorf("failed test for WriteRootfsRepo! Error: failed to read sources.list")
	}
	if sourceRepo+"\n" != string(sourcesData) {
		t.Errorf("failed test for WriteRootfsRepo! Error: read data not right!")
	}

	if ret, err := RemovePath(rootfsPath); err != nil || !ret {
		t.Errorf("failed test for WriteRootfsRepo! Error: failed to remove rootfs")
	}

}
