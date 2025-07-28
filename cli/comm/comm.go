/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package comm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	StatesJson     = "/var/lib/linglong/states.json"
)

type Options struct {
	Workdir string
	Config  string
}

type Source struct {
	Kind    string
	Digest  string
	Url     string
	Commit  string
	Version string
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

// 玲珑仓库的定义的架构和 os 获取系统架构对应关系
// os 获取架构为 amd64，玲珑仓库里定义为x86_64，
// 但是 arm64 在玲珑仓库定义 arm64，而没有使用 aarch64 需要该函数进行转换
// loong64 对应 loongarch64
func ArchConvert(arch string) string {
	switch arch {
	case "amd64":
		return "x86_64"
	case "loong64":
		return "loongarch64"
	default:
		return arch
	}
}

// 对生成的 Source 数组进行去重
func RemoveExcessDeps(sources []Source) []Source {
	var result []Source
	uniqueMap := make(map[string]bool)
	for _, pkg := range sources {
		key, _ := json.Marshal(pkg)
		// 如果 key 不存在于 map 中，则添加
		if _, ok := uniqueMap[string(key)]; !ok {
			uniqueMap[string(key)] = true
			result = append(result, pkg)
		}
	}
	return result
}

// 对buildext中depends/build_depends数组去重，去空白，去空项
func RemoveExcessDepends(depends []string) []string {
	m := make(map[string]struct{})
	var result []string
	for _, v := range depends {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, exists := m[v]; !exists {
			m[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}
