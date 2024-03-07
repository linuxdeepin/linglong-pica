/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package comm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

const (
	PicaConfigDir  = ".pica"
	PicaConfigJson = "config.json"
	packageYaml    = "package.yaml"
	LinglongYaml   = "linglong.yaml"
	Workdir        = "linglong-pica"
	PackageDir     = "package"
	AptlyDir       = ".aptly"
	LlSourceDir    = "linglong/sources"
)

type Options struct {
	Workdir string
	Config  string
}

func ExecAndWait(timeout int, name string, arg ...string) (stdout, stderr string, err error) {
	log.Logger.Debugf("cmd: %s %+v\n", name, arg)
	cmd := exec.Command(name, arg...)
	var bufStdout, bufStderr bytes.Buffer
	cmd.Stdout = &bufStdout
	cmd.Stderr = &bufStderr
	err = cmd.Start()
	if err != nil {
		err = fmt.Errorf("start fail: %w", err)
		return
	}

	// wait for process finished
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		if err = cmd.Process.Kill(); err != nil {
			err = fmt.Errorf("timeout: %w", err)
			return
		}
		<-done
		err = fmt.Errorf("time out and process was killed")
	case err = <-done:
		stdout = bufStdout.String()
		stderr = bufStderr.String()
		if err != nil {
			err = fmt.Errorf("run: %w", err)
			return
		}
	}
	return
}

func BuildPackPath(work string) string {
	return filepath.Join(work, PackageDir)
}

// ll-pica 工具，配置目录
func PicaConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), PicaConfigDir)
}

// 初始化 ll-pica 配置目录
func InitPicaConfigDir() {
	// 创建 ~/.pica 目录
	if ret, _ := fs.CheckFileExits(PicaConfigPath()); !ret {
		if exited, err := fs.CreateDir(PicaConfigPath()); !exited {
			log.Logger.Errorf("create picadir %s: failed: %s", PicaConfigPath(), err)
		}
	} else {
		log.Logger.Infof("picadir is exited %s", PicaConfigPath())
	}
}

// ll-pica 工具的配置 json 文件
func PicaConfigJsonPath() string {
	return filepath.Join(os.Getenv("HOME"), PicaConfigDir, PicaConfigJson)
}

// ll-pica 工作目录
func WorkPath(path string) string {
	var (
		err      error
		workPath string
	)
	if path == "" {
		workPath = filepath.Join(os.Getenv("HOME"), ".cache", Workdir)
	} else {
		workPath, err = filepath.Abs(path)
		if err != nil {
			log.Logger.Errorf("Trans %s err: %s ", path, err)
		} else {
			log.Logger.Infof("workdir path: %s", workPath)
		}
	}
	return workPath
}

func InitWorkDir(workPath string) {
	// 创建 workdir
	if ret, _ := fs.CheckFileExits(workPath); !ret {
		if exited, err := fs.CreateDir(workPath); !exited {
			log.Logger.Errorf("create workdir %s: failed: %s", workPath, err)
		}
	} else {
		log.Logger.Infof("workdir is exited %s", workPath)
	}
}

func ConfigFilePath(work string, config string) string {
	var (
		configFilePath string
		err            error
	)

	if config == "" {
		configFilePath = filepath.Join(work, packageYaml)
	} else {
		if configFilePath, err = filepath.Abs(config); err != nil {
			log.Logger.Errorf("Trans %s err: %s ", configFilePath, err)
		} else {
			log.Logger.Infof("Trans success path: %s", configFilePath)
		}
	}

	return configFilePath
}

// aptly 缓存路径
func AptlyCachePath() string {
	return filepath.Join(os.Getenv("HOME"), AptlyDir)
}

// 返回 linglong.yaml 中定义的 deb 包缓存路径
func LLSourcePath(path string) string {
	return filepath.Join(path, LlSourceDir)
}
