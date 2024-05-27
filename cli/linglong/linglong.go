/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package linglong

import (
	"fmt"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/cli/deb"
	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

type LinglongBuilder struct {
	Package    Package      `yaml:"package"`
	Base       string       `yaml:"base"`
	Runtime    string       `yaml:"runtime"`
	Command    []string     `yaml:"command"`
	Sources    []deb.Source `yaml:"sources"`
	Build      []string     `yaml:"-"`
	BuildInput string       `yaml:"build"` // 用来接收build字段，从yaml文件读入的值
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
  {{- range $line := .Build}}
  {{- printf "\n  %s" $line}}
  {{- end}}
`

func NewLinglongBuilder() *LinglongBuilder {
	return &LinglongBuilder{}
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

func (ts *LinglongBuilder) LinglongExport(path string) bool {
	// caller ll-builder export --local
	if ret, msg, err := comm.ExecAndWait(1<<20, "sh", "-c",
		fmt.Sprintf("cd %s && ll-builder export", path)); err != nil {
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
