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
	"fmt"
	"html/template"
	. "ll-pica/core/comm"
	. "ll-pica/core/elf"
	. "ll-pica/utils/fs"
	. "ll-pica/utils/log"
	"os"
	"strings"
)

// var Logger *zap.SugaredLogger

// func init() {
// 	Logger = InitLog()
// }

type BinFormatReactor struct {
	SearchPath      string   // search dir
	PathFilter      []string // filter with Search result
	ElfLDDPath      map[string]uint
	ElfNeedPath     map[string]uint
	ElfEntrySoPath  map[string]uint
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
 * @brief FixElfNeedPath
 * @param exclude []string 排除的文件的列表
 * @return true
 */
func (ts *BinFormatReactor) FixElfNeedPath(exclude []string) bool {

	for _, exStr := range exclude {
		if len(exStr) > 0 {
			deleteKeyList := FilterMap(ts.ElfNeedPath, func(str string) bool {
				return strings.HasSuffix(str, exStr)
			})

			if len(deleteKeyList) > 0 {
				for _, v := range deleteKeyList {
					delete(ts.ElfNeedPath, v)
				}
			}
		}
	}
	return true
}

/*!
 * @brief CopyElfNeedPath
 * @param prefix, dst string
 * @return true
 */
func (ts *BinFormatReactor) CopyElfNeedPath(prefix, dst string) bool {
	if len(prefix) <= 0 || len(ts.ElfNeedPath) <= 0 {
		return false
	}

	for v := range ts.ElfNeedPath {
		srcPath := prefix + "/" + v
		if ret, _ := CheckFileExits(srcPath); ret {
			var dstPath string
			if strings.HasPrefix(v, "/usr/lib") {
				dstPath = dst + strings.Replace(v, "/usr/lib", "/lib", 1)
			} else {
				dstPath = dst + v
			}

			dstParentPath := GetFilePPath(dstPath)
			Logger.Debugf("Copying path %s ", dstParentPath)
			if ret, _ := CheckFileExits(dstParentPath); !ret {
				CreateDir(dstParentPath)
			}
			if err := CopyFileKeepPermission(srcPath, dstPath, true, true); err != nil {
				Logger.Warnf("copy file failed %v", err)
				continue
			}
		}
		Logger.Debugf("Copying src path %s not found", srcPath)
		continue
	}
	// CopyFileKeepPermission()
	return true
}

/*!
 * @brief GetElfList
 * @param exclude string 排除的目录
 * @return 返回elf列表
 */
func (ts *BinFormatReactor) GetElfList(exclude string) bool {
	Logger.Debugf("get find elf miss depends: ", ts.SearchPath, "exclude: ", exclude)
	elf_binary_path, err := GetElfWithPath(ts.SearchPath)
	if err != nil {
		Logger.Debugf("get elf with path failed! %s", err)
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
		//Logger.Debugf("filter resut: ", filterResut)
		for _, v := range filterResut {
			ts.ElfLDDPath[v] = 1
		}

		return true
	}

	return false
}

// GetEntryDlopenList 通过Entry的入口，判断elf文件中Dlopen的依赖清单
func (ts *BinFormatReactor) GetEntryDlopenList(exclude []string) bool {
	//
	IsNotIncluded := func(filename string) bool {
		for _, v := range exclude {
			if strings.HasSuffix(v, filename) {
				return true
			}
		}
		return false
	}

	IsHaveDlopen := func(filename string) bool {
		// strings /usr/bin/deepin-movie| grep -i dlopen
		cmd := fmt.Sprintf("strings %s | grep -q dlopen", filename)
		if msg, ret, err := ExecAndWait(10, "bash", "-c", cmd); err != nil {
			Logger.Debugf("check dlopen failed: %v", err, msg, ret)
			return false
		} else {
			return true
		}
	}

	Logger.Debugf("get had entry elf list: ", ts.SearchPath, "exclude: ", exclude)

	if len(ts.ElfLDDPath) == 0 {
		Logger.Warn("Have not elf list??")
		return false
	}

	elf_have_entry_list := FilterMap(ts.ElfLDDPath, func(str string) bool {
		return !IsNotIncluded(str) && IsElfEntry(str) && IsHaveDlopen(str)
	})

	if len(elf_have_entry_list) == 0 {
		Logger.Warnf("have not search include entry elf file with:", ts.SearchPath)
		return false
	}

	ts.ElfEntrySoPath = make(map[string]uint)
	for _, v := range elf_have_entry_list {
		Logger.Debugf("process path: %s", v)
		if ret, err := GetDlopenDepends(v); err != nil {
			continue
		} else {
			Logger.Debugf("%v", ret)
			if len(ret) == 0 {
				continue
			} else {
				entry_dlopen_so := Filter(ret, func(str string) bool {
					return !IsNotIncluded(str)
				})

				for _, idx := range entry_dlopen_so {
					ts.ElfEntrySoPath[idx] = 1
				}
			}
		}
	}

	return false
}

type ElfLDDShellTemplate struct {
	ELFNameString    string
	DlopenNameString []string
	OutputNameString string
	Verbose          bool
}

// fixme: ldd not found case
const TMPL_ELF_LDD = `#!/bin/bash
set -x

ldconfig -p > /tmp/libcache.db

{{range $idx, $element := .DlopenNameString}}
DLOPEN_SOPATH=$(cat /tmp/libcache.db | grep {{ $element }} | awk '{print $4}'|head -n 1)
[[ -f ${DLOPEN_SOPATH} ]] && (echo ${DLOPEN_SOPATH} >> /tmp/elfsonamelist)
[[ -f ${DLOPEN_SOPATH} ]] && (ldd ${DLOPEN_SOPATH} | awk '{print $3}' | sort| uniq | sed '/^$/d' >> /tmp/elfsonamelist)
{{end}}
{{ if len .ELFNameString }}
ldd {{.ELFNameString}} | awk '{print $3}' | sort| uniq | sed '/^$/d' >> /tmp/elfsonamelist
{{end}}

{{ if len .OutputNameString}}
touch {{.OutputNameString}}
{{end}}

[[ -f /tmp/elfsonamelist ]] && (cat /tmp/elfsonamelist | sort | uniq | sed '/^$/d' >>  {{.OutputNameString}})

rm -v /tmp/libcache.db
rm -v /tmp/elfsonamelist

echo end
`

/*!
 * @brief RenderElfWithLDD
 * @param output output file
 * @param save save file
 * @return bool, error
 */
func (ts *BinFormatReactor) RenderElfWithLDD(output, save string) (bool, error) {

	// init template
	Logger.Debug("render elf with ldd : ", ts.SearchPath)
	tpl, err := template.New("elfldd").Parse(TMPL_ELF_LDD)

	if err != nil {
		Logger.Fatalf("parse deb shell template failed! ", err)
		return false, nil
	}

	elfLDDShell := ElfLDDShellTemplate{"", make([]string, 0), output, false}

	for elfStr := range ts.ElfLDDPath {
		elfLDDShell.ELFNameString += elfStr
		elfLDDShell.ELFNameString += " "
	}

	for elfStr := range ts.ElfEntrySoPath {
		elfLDDShell.DlopenNameString = append(elfLDDShell.DlopenNameString, elfStr)
	}

	// create save file
	Logger.Debug("create save file: ", save)
	saveFd, ret := os.Create(save)
	if ret != nil {
		Logger.Fatalf("save to %s failed!", save)
		return false, nil
	}
	defer saveFd.Close()

	// render template
	// Logger.Debug("render template: ", elfLDDShell)
	tpl.Execute(saveFd, elfLDDShell)

	return true, nil
}

// func (ts *BinFormatReactor) GetElfWithLDD(exclude string) []string {
// 	if len(ts.ElfLDDPath) > 0 {

//		}
//		return nil
//	}
func GetDlopenDepends(path string) ([]string, error) {
	// strings /bin/bash | grep  "\.so"
	cmd := fmt.Sprintf("strings %s | grep \\\\.so ", path)
	if msg, ret, err := ExecAndWait(10, "bash", "-c", cmd); err != nil {
		Logger.Debugf("check elf entry failed: %v", err, msg, ret)
		return nil, err
	} else {
		return strings.Split(msg, "\n"), nil
	}
	// return nil, fmt.Errorf("not found")
}
func GetElfNeedWithLDD(elfSearchDir string) (string, error) {
	Logger.Debug("get elf need with ldd: ", elfSearchDir)
	return "", nil
}

func GetFindElfMissDepends(elfSearchDir string) (bool, error, []string) {

	// find . -type f  -exec file {} \; | grep  ELF | awk -F: '{print $1}' | xargs -I{} ldd {} | grep -i "not found"

	Logger.Debug("get find elf miss depends: ", elfSearchDir)
	elf_binary_path, err := GetElfWithPath(elfSearchDir)
	if err != nil {
		Logger.Debugf("get elf with path failed! %s", err)
	}
	Logger.Debug("elf binary path: ", elf_binary_path)
	// fixme:(heysion) fix get elf binary path with depend list
	return false, nil, nil

}

func GetElfNeedWithStrace(elf string) (string, error) {
	Logger.Debug("get elf need with strace: ", elf)
	return "", nil
}

func ChrootExecShell(chrootDirPath, shell string, bindMounts []string) (bool, string, error) {
	Logger.Debugf("chroot exec shell: %s shell: %s", chrootDirPath, shell)

	// fixme:(heysion) mount /mnt/workdir/debdir/ to chroot /mnt/workdir/debdir
	if len(bindMounts) > 0 {
		for _, srcPath := range bindMounts {
			dstPath := chrootDirPath + srcPath
			CreateDir(dstPath)
			Logger.Debug("bind mount: ", srcPath, dstPath)
			// bind mount src to dst
			if _, msg, err := ExecAndWait(10, "mount", "-B", srcPath, dstPath); err != nil {
				Logger.Fatalf("mount %s to %s failed! ", srcPath, dstPath, err, msg)
			}
			// defer func() { RemovePath(dstPath) }()
			defer func() { Logger.Debugf("remove %s", dstPath) }()
			defer func() { UmountPath(dstPath) }()
			defer func() { Logger.Debugf("umount %s", dstPath) }()
		}

	}

	// mount shell to chroot
	shellSrcPath := GetFilePPath(shell)
	shellDstPath := chrootDirPath + shellSrcPath
	shellChrootPath := chrootDirPath + shell

	Logger.Debugf("shell src path: %s to %s", shellSrcPath, shellDstPath)
	if ret, _ := CheckFileExits(shellDstPath); !ret {
		CreateDir(shellDstPath)
	}

	if _, msg, err := ExecAndWait(10, "mount", "-B", shellSrcPath, shellDstPath); err != nil {
		Logger.Fatalf("mount %s to %s failed! ", shell, shellDstPath, err, msg)
		return false, msg, err
	}

	// CreateDir(shellDstPath)
	// defer func() { RemovePath(shellDstPath) }()
	defer func() { Logger.Debugf("remove %s", shellDstPath) }()

	defer func() { UmountPath(shellDstPath) }()
	defer func() { Logger.Debugf("umount %s", shellDstPath) }()

	// chmod +x shell
	if _, msg, err := ExecAndWait(10, "chmod", "+x", "-R", shellChrootPath); err != nil {
		Logger.Fatalf("chmod +x %s failed! ", shellChrootPath, err, msg)
		return false, msg, err
	}

	// chroot shell
	Logger.Debugf("chroot shell: path: %s shell:%s", chrootDirPath, shell)
	if ret, msg, err := ExecAndWait(4096, "chroot", chrootDirPath, shell); err != nil {
		Logger.Fatalf("chroot exec shell failed! ", err, msg, ret)
		return false, msg, err
	} else {
		Logger.Debugf("chroot exec shell msg:", ret, msg)
	}
	return true, "", nil
}

type DebShellTemplate struct {
	ExtraPackageStr string
	DebString       string
	PreCommand      []string
	PostCommand     []string
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

function pre_command {
	{{range $idx, $element := .PreCommand}}
	{{$element}}
	{{end}}
	echo OK
}

function post_command {
	{{range $idx, $element := .PostCommand}}
	{{$element}}
	{{end}}
	echo OK
}
{{if len .PreCommand }}pre_command{{end}}
apt_update
{{if len .DebString }}apt_install_deb {{end}}
{{if len .ExtraPackageStr }}apt_install_pkgs{{end}}
{{if len .PostCommand }}post_command{{end}}
`

func RenderDebConfig(DebConf DebConfig, save string) (bool, error) {

	// init template
	// Logger.Debug("render deb config: ", DebConf)
	tpl, err := template.New("pica").Parse(DEB_SHELL_TMPL)

	if err != nil {
		Logger.Fatalf("parse deb shell template failed! ", err)
		return false, nil
	}

	debShell := DebShellTemplate{"", "", make([]string, 0), make([]string, 0), false}

	for _, debStr := range DebConf.FileElement.Deb {

		// Logger.Debugf("deb str: %s path :%s", debStr, debStr.Path)
		debShell.DebString += debStr.Path
		debShell.DebString += " "
	}

	if len(DebConf.FileElement.Package) != 0 {
		debShell.ExtraPackageStr = strings.Join(DebConf.FileElement.Package, " ")
	} else {
		debShell.ExtraPackageStr = ""
	}

	// PreCommand
	if len(DebConf.ChrootInfo.PreCmd) > 0 {
		for _, cmd := range DebConf.ChrootInfo.PreCmd {
			debShell.PreCommand = append(debShell.PreCommand, cmd)
		}
	}
	// PostCommand
	if len(DebConf.ChrootInfo.PostCmd) > 0 {
		for _, cmd := range DebConf.ChrootInfo.PostCmd {
			debShell.PostCommand = append(debShell.PostCommand, cmd)
		}
	}

	// create save file
	Logger.Debug("create save file: ", save)
	saveFd, ret := os.Create(save)
	if ret != nil {
		Logger.Fatalf("save to %s failed!", save)
		return false, nil
	}
	defer saveFd.Close()

	// render template
	Logger.Debug("render template: ", debShell)
	tpl.Execute(saveFd, debShell)

	return true, nil
}
