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
	"html/template"
	. "ll-pica/core/comm"
	. "ll-pica/utils/fs"
	. "ll-pica/utils/log"
	"os"
)

type LinglongBuder struct {
	Appid       string
	Version     string
	Runtime     string
	Rversion    string
	Description string
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
