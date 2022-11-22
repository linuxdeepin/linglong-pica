/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package comm

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"pkg.deepin.com/linglong/pica/cmd/ll-pica/utils/fs"
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
	if ret, err := fs.CreateDir("/tmp/ll-rootfs/etc/apt"); !ret && err != nil {
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

	if ret, err := fs.RemovePath(rootfsPath); err != nil || !ret {
		t.Errorf("failed test for WriteRootfsRepo! Error: failed to remove rootfs")
	}

}

// FixCachePath
var testDataFixCachePath = []struct {
	workdir   string
	cachePath string
}{
	{"/tmp/workdir1", "/mnt/workdir/cache.yaml"},
	{"/mnt/workdir", "/mnt/workdir/cache.yaml"},
	{"/tmp/workdir2", "/tmp/linglong/cache.yaml"},
}

func TestFixCachePath(t *testing.T) {
	// 测试使用了 -w ,没用 -f场景
	config := Config{
		Workdir:   testDataFixCachePath[0].workdir,
		CachePath: testDataFixCachePath[0].cachePath,
	}
	// workdir不存在时
	if ret, err := config.FixCachePath(); ret || err == nil {
		t.Errorf("Failed test for TestFixCache! Error : %+v", err)
	}
	// 新建目录
	if ret, err := fs.CreateDir(config.Workdir); !ret || err != nil {
		t.Errorf("Failed test for TestFixCache! Error : failed to create dir %+v", config.Workdir)
	}
	defer fs.RemovePath(config.Workdir)
	if ret, err := config.FixCachePath(); !ret || err != nil {
		t.Errorf("Failed test for TestFixCache! Error :  %+v", err)
	}

	fixCachePath := config.Workdir + "/cache.yaml"
	if fixCachePath != config.CachePath {
		t.Errorf("Failed test for TestFixCache! Error :  fix cache path failed. ")
	}

	// 测试没用 -w -f 参数场景(需要root才能新建目录测试)
	// config = Config{
	// 	Workdir:   testDataFixCachePath[1].workdir,
	// 	CachePath: testDataFixCachePath[1].cachePath,
	// }
	// // workdir不存在时
	// if ret, err := config.FixCachePath(); ret || err == nil {
	// 	t.Errorf("Failed test for TestFixCache! Error : %+v", err)
	// }
	// // 新建目录
	// if ret, err := CreateDir(config.Workdir); !ret || err != nil {
	// 	t.Errorf("Failed test for TestFixCache! Error : failed to create dir %+v", config.Workdir)
	// }
	// defer RemovePath(config.Workdir)
	// if ret, err := config.FixCachePath(); !ret || err != nil {
	// 	t.Errorf("Failed test for TestFixCache! Error :  %+v", err)
	// }

	// fixCachePath = config.Workdir + "/cache.yaml"
	// if fixCachePath != config.CachePath {
	// 	t.Errorf("Failed test for TestFixCache! Error :  fix cache path failed. ")
	// }

	// 测试使用 -w -f 参数场景
	config = Config{
		Workdir:   testDataFixCachePath[2].workdir,
		CachePath: testDataFixCachePath[2].cachePath,
	}
	// workdir不存在时
	if ret, err := config.FixCachePath(); ret || err == nil {
		t.Errorf("Failed test for TestFixCache! Error : %+v", err)
	}
	// 新建目录
	if ret, err := fs.CreateDir(config.Workdir); !ret || err != nil {
		t.Errorf("Failed test for TestFixCache! Error : failed to create dir %+v", config.Workdir)
	}
	defer fs.RemovePath(config.Workdir)
	if ret, err := config.FixCachePath(); !ret || err != nil {
		t.Errorf("Failed test for TestFixCache! Error :  %+v", err)
	}

	if testDataFixCachePath[2].cachePath != config.CachePath {
		t.Errorf("Failed test for TestFixCache! Error :  fix cache path failed. ")
	}
}

func TestGetRefName(t *testing.T) {
	refs := "https://mirrors.ustc.edu.cn/deepin/pool/main/d/deepin-calculator/deepin-calculator_1:5.7.20-1_amd64.deb"
	t.Logf(refs)

	t.Logf(filepath.Base(refs))
}
