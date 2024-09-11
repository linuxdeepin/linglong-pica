/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package linglong

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

type LinglongBuilder struct {
	Package    Package       `yaml:"package"`
	Base       string        `yaml:"base"`
	Runtime    string        `yaml:"runtime"`
	Command    []string      `yaml:"command"`
	Sources    []comm.Source `yaml:"sources"`
	Build      []string      `yaml:"-"`
	BuildInput string        `yaml:"build"` // 用来接收build字段，从yaml文件读入的值
}

type LinglongCli struct {
	Arch    []string
	Channel string
	Version string
}

type Package struct {
	Appid       string `yaml:"id"`
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Kind        string `yaml:"kind"`
	Description string `yaml:"description"`
}

const LinglongBuilderTMPL = `version: "1"

package:
  id: {{.Package.Appid}}
  name: {{.Package.Name}}
  version: {{.Package.Version}}
  kind: {{.Package.Kind}}
  description: |
    {{.Package.Description}}

base: {{.Base}}
runtime: {{.Runtime}}

command:
  {{- range $line := .Command}}
  {{- printf "\n  - %s" $line}}
  {{- end}}
{{if .Sources}}
sources:
{{- range .Sources}}
  - kind: {{.Kind}}
    url: {{.Url}}
    digest: {{.Digest}}
{{end}}
{{- end}}
build: |
  set -x
  {{- range $line := .Build}}
  {{- printf "\n  %s" $line}}
  {{- end}}
`

func NewLinglongBuilder() *LinglongBuilder {
	return &LinglongBuilder{}
}

func NewLinglongCli() *LinglongCli {
	return &LinglongCli{}
}

// create linglong.yaml
func (ts *LinglongBuilder) CreateLinglongYaml(path string) bool {

	tpl, err := template.New("linglong").Parse(LinglongBuilderTMPL)

	if err != nil {
		log.Logger.Fatalf("parse deb shell template failed! ", err)
		return false
	}

	// create save file
	log.Logger.Debug("create save file: ", path)
	saveFd, ret := os.Create(path)
	if ret != nil {
		log.Logger.Fatalf("save to %s failed!", path)
		return false
	}
	defer saveFd.Close()

	// render template
	log.Logger.Debug("render template: ", ts)
	tpl.Execute(saveFd, ts)

	return true

}

// read linglong.yaml
func (ts *LinglongBuilder) ReadLinglongYaml(path string) bool {
	log.Logger.Infof("load %s", path)
	llYamlFd, err := os.ReadFile(path)
	if err != nil {
		log.Logger.Errorf("load %s error: %v", path, err)
	} else {
		if err = yaml.Unmarshal(llYamlFd, ts); err != nil {

			log.Logger.Errorf("unmarshal %s error: %v", path, err)
		}
		return true
	}
	return false
}

// build linglong.yaml
func (ts *LinglongBuilder) CreateLinglongBuilder(path string) bool {

	log.Logger.Debugf("create save file: ", path)

	// check workstation
	if ret, err := fs.CheckFileExits(path); err != nil && !ret {
		log.Logger.Errorf("workstation witch convert not found: %s", path)
		return false
	} else {
		err := os.Chdir(path)
		if err != nil {
			log.Logger.Errorf("workstation can not enter directory: %s", path)
			return false
		}
	}

	// caller ll-builder build
	if ret, msg, err := comm.ExecAndWait(10, "ll-builder", "build"); err != nil {
		log.Logger.Fatalf("ll-builder failed: ", err, msg, ret)
		return false
	} else {
		log.Logger.Infof("ll-builder succeeded: ", path, ret)
		return true
	}
}

// 调用 ll-builder build
func (ts *LinglongBuilder) LinglongBuild(path string) bool {
	if ret, msg, err := comm.ExecAndWait(300, "sh", "-c",
		fmt.Sprintf("cd %s && ll-builder build", path)); err != nil {
		log.Logger.Fatalf("msg: %+v err:%+v, out: %+v", msg, err, ret)
	} else {
		log.Logger.Infof("msg: %+v err:%+v, out: %+v", msg, err, ret)
	}
	return true
}

func (ts *LinglongBuilder) LinglongExport(path string, exportFile string) bool {
	runCmd := "ll-builder export"
	if exportFile == "layer" {
		runCmd += " --layer"
	}
	// caller ll-builder export --local
	if ret, msg, err := comm.ExecAndWait(1<<20, "sh", "-c",
		fmt.Sprintf("cd %s && %s", path, runCmd)); err != nil {
		log.Logger.Fatalf("msg: %+v err:%+v, out: %+v", msg, err, ret)
	} else {
		log.Logger.Infof("%s export success.", path)
	}

	// chmod 755 uab
	// if bundleList, err := fs.FindBundlePath(appExportPath); err != nil {
	// 	log.Logger.Errorf("not found bundle")
	// 	return false
	// } else {
	// 	for _, bundle := range bundleList {
	// 		log.Logger.Infof("chmod 0755 for %s", bundle)
	// 		if err := os.Chmod(bundle, 0755); err != nil {
	// 			log.Logger.Errorf("chmod 0755 for %s failed！", bundle)
	// 			return false
	// 		}
	// 	}
	// }
	return true
}

func (cli *LinglongCli) LinglongCliInfo(appid string) {
	if ret, msg, err := comm.ExecAndWait(10, "sh", "-c",
		fmt.Sprintf("ll-cli info %s", appid)); err != nil {
		log.Logger.Warnf("ll-cli info error: %s", msg)
	} else {
		err = json.Unmarshal([]byte(ret), &cli)
		if err != nil {
			log.Logger.Errorf("unmarshal error: %s", err)
		}
	}
}

// 获取 base 里面安装的包列表
func (cli *LinglongCli) GetBaseInsPack() []string {
	var packages []string

	// 读取 pica 的配置
	config := comm.NewConfig()
	config.ReadConfigJson()

	cli.LinglongCliInstall(config.BaseId, config.Version)
	// 获取 base 的 info
	cli.LinglongCliInfo(config.BaseId)
	if ret, msg, err := comm.ExecAndWait(60, "sh", "-c",
		fmt.Sprintf("cat /var/lib/linglong/layers/%s/%s/%s/%s/runtime/files/var/lib/dpkg/status | awk -F': ' '/^Package: /{a=a\",\"$2} END{sub(/^,/,\"\",a);printf a}'",
			cli.Channel, config.BaseId, cli.Version, cli.Arch[0])); err != nil {
		log.Logger.Warnf("cat dpkg/status error: %s", msg)
	} else {
		packages = append(packages, strings.Split(ret, ",")...)
	}
	return packages
}

// 获取 runtime 里面安装的包列表
func (cli *LinglongCli) GetRuntimeInsPack() []string {
	var packages []string

	// 读取 pica 的配置
	config := comm.NewConfig()
	config.ReadConfigJson()

	cli.LinglongCliInstall(config.Id, config.Version)
	// 获取 runtime 的 info
	cli.LinglongCliInfo(config.Id)
	if ret, msg, err := comm.ExecAndWait(60, "sh", "-c",
		fmt.Sprintf("cat /var/lib/linglong/layers/%s/%s/%s/%s/runtime/files/packages.list | awk -F': ' '/^Package: /{a=a\",\"$2} END{sub(/^,/,\"\",a);printf a}'",
			cli.Channel, config.Id, cli.Version, cli.Arch[0])); err != nil {
		log.Logger.Warnf("cat runtime/package.list error: %s", msg)
	} else {
		packages = append(packages, strings.Split(ret, ",")...)
	}
	return packages
}

func (cli *LinglongCli) LinglongCliInstall(appid, version string) {
	if ret, _, err := comm.ExecAndWait(1<<20, "sh", "-c",
		fmt.Sprintf("ll-cli install %s/%s", appid, version)); err != nil {
		log.Logger.Infof("out: %+v", ret)
	}
}
