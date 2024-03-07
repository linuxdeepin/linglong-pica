/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package linglong

import (
	"os"
	"text/template"

	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/cli/deb"
	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

type LinglongBuder struct {
	Appid       string
	Name        string
	Version     string
	Base        string
	Runtime     string
	Rversion    string
	Description string
	Command     string
	Sources     []deb.Source
	Build       []string
}

const LinglongBuilderTMPL = `version: "1"

package:
  id: {{.Appid}}
  name: {{.Name}}
  version: {{.Version}}
  kind: app
  description: |
    {{.Description}}

base: {{.Base}}/{{.Rversion}}
runtime: {{.Runtime}}/{{.Rversion}}

command:
  - "{{.Command}}"

sources:
{{- range .Sources}}
  - kind: {{.Kind}}
    url: {{.Url}}
    digest: {{.Digest}}
{{end}}
build: |
  #>>> auto generate by ll-pica begin
  {{- range $line := .Build}}
  {{- printf "\n  %s" $line}}
  {{- end}}
  #>>> auto generate by ll-pica end
`

// CreateLinglongYamlBuilder
func (ts *LinglongBuder) CreateLinglongYamlBuilder(path string) bool {

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

// build linglong.yaml
func (ts *LinglongBuder) CreateLinglongBuilder(path string) bool {

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

func (ts *LinglongBuder) LinglongExport(path string) bool {
	log.Logger.Debugf("ll-builder import : ", ts.Appid)
	appExportPath := fs.GetFilePPath(path)
	appExportPath = fs.GetFilePPath(appExportPath)
	// check workstation
	if ret, err := fs.CheckFileExits(path); err != nil && !ret {
		log.Logger.Errorf("workstation witch convert not found: %s", path)
		return false
	} else {
		err := os.Chdir(appExportPath)
		if err != nil {
			log.Logger.Errorf("workstation can not enter directory: %s", appExportPath)
			return false
		}
	}
	// caller ll-builder export --local
	if ret, msg, err := comm.ExecAndWait(120, "ll-builder", "export", path); err != nil {
		log.Logger.Fatalf("ll-builder export failed: ", err, msg, ret)
		return false
	} else {
		log.Logger.Infof("ll-builder export succeeded: ", path, ret)
	}

	// chmod 755 uab
	if bundleList, err := fs.FindBundlePath(appExportPath); err != nil {
		log.Logger.Errorf("not found bundle")
		return false
	} else {
		for _, bundle := range bundleList {
			log.Logger.Infof("chmod 0755 for %s", bundle)
			if err := os.Chmod(bundle, 0755); err != nil {
				log.Logger.Errorf("chmod 0755 for %s failed！", bundle)
				return false
			}
		}
	}
	return true
}
