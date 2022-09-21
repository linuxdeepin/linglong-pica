/*
 * Copyright (c) 2022. Uniontech Software Ltd. All rights reserved.
 *
 * Author: Heysion Y. <heysion@deepin.com>
 *
 * Maintainer: Heysion Y. <heysion@deepin.com>
 *
 * SPDX-License-Identifier: GNU General Public License v3.0 or later
 */
package linglong

import (
	"encoding/json"
	"io/ioutil"
	. "ll-pica/core/comm"
	. "ll-pica/utils/fs"
	. "ll-pica/utils/log"
	"os"
	"text/template"
)

type LinglongBuder struct {
	Appid       string
	Version     string
	Runtime     string
	Rversion    string
	Description string
}

type RuntimeJson struct {
	Appid       string   `json:"appid"`
	Arch        []string `json:"arch"`
	Base        string   `json:"base"`
	Description string   `json:"description"`
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Runtime     string   `json:"runtime"`
	Version     string   `json:"version"`
}

// LoadRuntimeInfo
func (ts *LinglongBuder) LoadRuntimeInfo(path string) bool {
	// load runtime info from file
	if ret, err := CheckFileExits(path); !ret {
		Logger.Warnf("load runtime info failed: %v", err)
		return false
	}
	var runtimedir RuntimeJson
	runtimedirFd, err := ioutil.ReadFile(path)
	if err != nil {
		Logger.Errorf("get %s error: %v", path, err)
		return false
	}
	err = json.Unmarshal(runtimedirFd, &runtimedir)
	if err != nil {
		Logger.Errorf("error: %v", err)
		return false
	}
	// copy to LinglongBuder
	if runtimedir.Appid != "" && runtimedir.Version != "" {
		ts.Runtime = runtimedir.Appid
		ts.Rversion = runtimedir.Version
		return true
	}

	return false
}

const LinglongBuilderTMPL = `
package:
  id:  {{.Appid}}
  version: {{.Version}}
  kind: app
  description: |
    {{.Description}}

runtime:
  id: {{.Runtime}}
  version: {{.Rversion}}

source:
  kind: local

build:
  kind: pica
`

// CreateLinglongYamlBuilder
func (ts *LinglongBuder) CreateLinglongYamlBuilder(path string) bool {

	tpl, err := template.New("linglong").Parse(LinglongBuilderTMPL)

	if err != nil {
		Logger.Fatalf("parse deb shell template failed! ", err)
		return false
	}

	// create save file
	Logger.Debug("create save file: ", path)
	saveFd, ret := os.Create(path)
	if ret != nil {
		Logger.Fatalf("save to %s failed!", path)
		return false
	}
	defer saveFd.Close()

	// render template
	Logger.Debug("render template: ", ts)
	tpl.Execute(saveFd, ts)

	return true

}

// CreateLinglongBuilder
func (ts *LinglongBuder) CreateLinglongBuilder(path string) bool {

	Logger.Debugf("create save file: ", path)

	// check workstation
	if ret, _ := CheckFileExits(path); !ret {
		Logger.Errorf("workstation witch convert not found: %s", path)
		return false
	} else {
		err := os.Chdir(path)
		if err != nil {
			Logger.Errorf("workstation can not enter directory: %s", path)
			return false
		}
	}

	// caller ll-builder build
	if ret, msg, err := ExecAndWait(10, "ll-builder", "build"); err != nil {
		Logger.Fatalf("ll-builder failed: ", err, msg, ret)
		return false
	} else {
		Logger.Infof("ll-builder succeeded: ", path, ret)
		return true
	}
}

func (ts *LinglongBuder) LinglongExport(path string) bool {
	Logger.Debugf("ll-builder export : ", ts.Appid)
	appExportPath := GetFilePPath(path)
	// check workstation
	if ret, _ := CheckFileExits(path); !ret {
		Logger.Errorf("workstation witch convert not found: %s", path)
		return false
	} else {
		err := os.Chdir(appExportPath)
		if err != nil {
			Logger.Errorf("workstation can not enter directory: %s", appExportPath)
			return false
		}
	}
	// caller ll-builder export --local
	if ret, msg, err := ExecAndWait(120, "ll-builder", "export", "--local"); err != nil {
		Logger.Fatalf("ll-builder export failed: ", err, msg, ret)
		return false
	} else {
		Logger.Infof("ll-builder export succeeded: ", path, ret)
		return true
	}
}
