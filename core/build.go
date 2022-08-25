/*
 * Copyright (c) 2022. Uniontech Software Ltd. All rights reserved.
 *
 * Author: Heysion Y. <heysion@deepin.com>
 *
 * Maintainer: Heysion Y. <heysion@deepin.com>
 *
 * SPDX-License-Identifier: GNU General Public License v3.0 or later
 */
package core

import (
	"html/template"
	. "ll-pica/core/comm"
	. "ll-pica/core/elf"
	. "ll-pica/utils/fs"
	. "ll-pica/utils/log"
	"os"
	"strings"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	logger = InitLog()
}

type BinFormatReactor struct {
	SearchPath      string   // search dir
	PathFilter      []string // filter with Search result
	ElfLDDPath      map[string]uint
	ElfNeedPath     map[string]uint
	DynExcludeList  string // so libary exclude list
	CheckWithChroot bool   // check with chroot
	CheckWithStrace bool   // check with strace
	StraceBinPath   string // strace command bin path with chroot
	CheckWithLdd    bool   // check with ldd
	CheckWithLtrace bool   // check with ltrace
}

// func (ts *BinFormatReactor) New(searchPath string) BinFormatReactor {
// 	ts.SearchPath = searchPath
// 	return *ts
// }

func Filter(vs []string, f func(string) bool) []string {
	retVal := make([]string, 0)
	for _, v := range vs {
		if f(v) && len(v) > 0 {
			retVal = append(retVal, v)
		}
	}
	return retVal
}

func FilterMap(vs map[string]uint, f func(string) bool) []string {
	retVal := make([]string, 0)
	for v := range vs {
		if f(v) && len(v) > 0 {
			retVal = append(retVal, v)
		}
		continue
	}
	return retVal
}

/*!
 * @brief FixElfLDDPath
 * @param exclude []string 排除的文件的列表
 * @return true
 */
func (ts *BinFormatReactor) FixElfLDDPath(exclude []string) bool {

	for _, exStr := range exclude {
		if len(exStr) > 0 {
			deleteKeyList := FilterMap(ts.ElfLDDPath, func(str string) bool {
				return strings.HasPrefix(str, exStr)
			})

			if len(deleteKeyList) > 0 {
				for _, v := range deleteKeyList {
					delete(ts.ElfLDDPath, v)
				}
			}
		}
	}
	return true
}

/*!
 * @brief GetElfList
 * @param exclude string 排除的目录
 * @return 返回elf列表
 */
func (ts *BinFormatReactor) GetElfList(exclude string) bool {
	logger.Debugf("get find elf miss depends: ", ts.SearchPath, "exclude: ", exclude)
	elf_binary_path, err := GetElfWithPath(ts.SearchPath)
	if err != nil {
		logger.Debugf("get elf with path failed! %s", err)
		return false
	}

	if len(elf_binary_path) > 0 {
		if len(exclude) == 0 {
			return false
		}
		filterResut := Filter(elf_binary_path, func(str string) bool {
			return !strings.HasPrefix(str, exclude)
		})
		if len(filterResut) > 0 && ts.ElfLDDPath == nil {
			ts.ElfLDDPath = make(map[string]uint)
		}
		//logger.Debugf("filter resut: ", filterResut)
		for _, v := range filterResut {
			ts.ElfLDDPath[v] = 1
		}

		return true
	}

	return false
}

type ElfLDDShellTemplate struct {
	ELFNameString    string
	OutputNameString string
	Verbose          bool
}

const TMPL_ELF_LDD = `#!/bin/bash
set -x
ldd {{.ELFNameString}} | awk '{print $3}' | sort| uniq > {{.OutputNameString}}
ldd {{.ELFNameString}} | awk '{print $3}' | sort| uniq > tt.log
`

/*!
 * @brief RenderElfWithLDD
 * @param output output file
 * @param save save file
 * @return bool, error
 */
func (ts *BinFormatReactor) RenderElfWithLDD(output, save string) (bool, error) {

	// init template
	logger.Debug("render elf with ldd : ", ts.SearchPath)
	tpl, err := template.New("elfldd").Parse(TMPL_ELF_LDD)

	if err != nil {
		logger.Fatalf("parse deb shell template failed! ", err)
		return false, nil
	}

	elfLDDShell := ElfLDDShellTemplate{"", output, false}

	for elfStr := range ts.ElfLDDPath {

		elfLDDShell.ELFNameString += elfStr
		elfLDDShell.ELFNameString += " "
	}

	// create save file
	logger.Debug("create save file: ", save)
	saveFd, ret := os.Create(save)
	if ret != nil {
		logger.Fatalf("save to %s failed!", save)
		return false, nil
	}
	defer saveFd.Close()

	// render template
	// logger.Debug("render template: ", elfLDDShell)
	tpl.Execute(saveFd, elfLDDShell)

	return true, nil
}

// func (ts *BinFormatReactor) GetElfWithLDD(exclude string) []string {
// 	if len(ts.ElfLDDPath) > 0 {

// 	}
// 	return nil
// }

func GetElfNeedWithLDD(elfSearchDir string) (string, error) {
	logger.Debug("get elf need with ldd: ", elfSearchDir)
	return "", nil
}

func GetFindElfMissDepends(elfSearchDir string) (bool, error, []string) {

	// find . -type f  -exec file {} \; | grep  ELF | awk -F: '{print $1}' | xargs -I{} ldd {} | grep -i "not found"

	logger.Debug("get find elf miss depends: ", elfSearchDir)
	elf_binary_path, err := GetElfWithPath(elfSearchDir)
	if err != nil {
		logger.Debugf("get elf with path failed! %s", err)
	}
	logger.Debug("elf binary path: ", elf_binary_path)
	// fixme:(heysion) fix get elf binary path with depend list
	return false, nil, nil

}

func GetElfNeedWithStrace(elf string) (string, error) {
	logger.Debug("get elf need with strace: ", elf)
	return "", nil
}

func ChrootExecShell(chroot, shell, datadir string) (bool, string, error) {
	logger.Debugf("chroot exec shell: %s shell: %s", chroot, shell)

	// fixme:(heysion) mount /mnt/workdir/debdir/ to chroot /mnt/workdir/debdir
	shellChroot := chroot + datadir
	CreateDir(shellChroot)
	defer func() { os.RemoveAll(shellChroot) }()

	// mount shell to chroot
	logger.Debug("copy shell to chroot: ", shell, shellChroot)
	if _, msg, err := ExecAndWait(10, "mount", "-B", GetFilePPath(shell), shellChroot); err != nil {
		logger.Fatalf("mount %s to %s failed! ", shell, shellChroot, err, msg)
	}

	defer func() { ExecAndWait(10, "umount", shellChroot) }()

	// chmod +x shell
	if _, msg, err := ExecAndWait(10, "chmod", "+x", "-R", shellChroot); err != nil {
		logger.Fatalf("chmod +x %s failed! ", shellChroot, err, msg)
	}

	// chroot shell
	logger.Debugf("chroot shell: path: %s shell:%s", chroot, shell)
	if _, msg, err := ExecAndWait(1000, "chroot", chroot, shell); err != nil {
		logger.Fatalf("chroot exec shell failed! ", err, msg)
	}
	return true, "", nil
}

type DebShellTemplate struct {
	ExtraPackageStr string
	DebString       string
	Verbose         bool
}

const DEB_SHELL_TMPL = `#!/bin/bash
{{if .Verbose }}set -x {{end}}
function apt_install_pkgs {
    DEBIAN_FRONTEND=noninteractive apt-get install -y {{.ExtraPackageStr}}
    DEBIAN_FRONTEND=noninteractive apt-get install -f -y
}

function apt_update {
    DEBIAN_FRONTEND=noninteractive apt-get update
}

function apt_install_deb {
    DEBIAN_FRONTEND=noninteractive apt-get install -y {{.DebString}}
    DEBIAN_FRONTEND=noninteractive apt-get install -f -y
}

apt_update
apt_install_deb
{{if len .ExtraPackageStr }}apt_install_pkgs{{end}}
`

func RenderDebConfig(DebConf DebConfig, save string) (bool, error) {

	// init template
	logger.Debug("render deb config: ", DebConf)
	tpl, err := template.New("pica").Parse(DEB_SHELL_TMPL)

	if err != nil {
		logger.Fatalf("parse deb shell template failed! ", err)
		return false, nil
	}

	debShell := DebShellTemplate{"", "", false}

	for _, debStr := range DebConf.FileElement.Deb {

		logger.Debugf("deb str: %s path :%s", debStr, debStr.Path)
		debShell.DebString += debStr.Path
		debShell.DebString += " "
	}

	if len(DebConf.FileElement.Package) != 0 {
		debShell.ExtraPackageStr = strings.Join(DebConf.FileElement.Package, " ")
	} else {
		debShell.ExtraPackageStr = ""
	}

	// create save file
	logger.Debug("create save file: ", save)
	saveFd, ret := os.Create(save)
	if ret != nil {
		logger.Fatalf("save to %s failed!", save)
		return false, nil
	}
	defer saveFd.Close()

	// render template
	logger.Debug("render template: ", debShell)
	tpl.Execute(saveFd, debShell)

	return true, nil
}
