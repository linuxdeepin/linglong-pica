/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package core

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"pkg.deepin.com/linglong/pica/cmd/ll-pica/core/comm"
	"pkg.deepin.com/linglong/pica/cmd/ll-pica/core/elf"
	"pkg.deepin.com/linglong/pica/cmd/ll-pica/utils/fs"
	"pkg.deepin.com/linglong/pica/cmd/ll-pica/utils/log"
)

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
		if ret, err := fs.CheckFileExits(srcPath); err != nil && ret {
			var dstPath string
			if strings.HasPrefix(v, "/usr/lib") {
				dstPath = dst + strings.Replace(v, "/usr/lib", "/lib", 1)
			} else {
				dstPath = dst + v
			}

			dstParentPath := fs.GetFilePPath(dstPath)
			log.Logger.Debugf("Copying path %s ", dstParentPath)
			if ret, err := fs.CheckFileExits(dstParentPath); err != nil && !ret {
				fs.CreateDir(dstParentPath)
			}
			if err := fs.CopyFileKeepPermission(srcPath, dstPath, true, true); err != nil {
				log.Logger.Warnf("copy file failed %v", err)
				continue
			}
		}
		log.Logger.Debugf("Copying src path %s not found", srcPath)
		continue
	}
	// fs.CopyFileKeepPermission()
	return true
}

/*!
 * @brief GetElfList
 * @param exclude string 排除的目录
 * @return 返回elf列表
 */
func (ts *BinFormatReactor) GetElfList(exclude string) bool {
	log.Logger.Debugf("get find elf miss depends: ", ts.SearchPath, "exclude: ", exclude)
	elf_binary_path, err := elf.GetElfWithPath(ts.SearchPath)
	if err != nil {
		log.Logger.Debugf("get elf with path failed! %s", err)
		return false
	}

	if len(elf_binary_path) > 0 {
		if len(exclude) == 0 {
			return false
		}
		filterResult := Filter(elf_binary_path, func(str string) bool {
			return !strings.HasPrefix(str, exclude)
		})
		if len(filterResult) > 0 && ts.ElfLDDPath == nil {
			ts.ElfLDDPath = make(map[string]uint)
		}
		for _, v := range filterResult {
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
		if msg, ret, err := comm.ExecAndWait(10, "bash", "-c", cmd); err != nil {
			log.Logger.Debugf("check dlopen failed: %v", err, msg, ret)
			return false
		} else {
			return true
		}
	}

	log.Logger.Debugf("get had entry elf list: ", ts.SearchPath, "exclude: ", exclude)

	if len(ts.ElfLDDPath) == 0 {
		log.Logger.Warn("Have not elf list??")
		return false
	}

	elf_have_entry_list := FilterMap(ts.ElfLDDPath, func(str string) bool {
		return !IsNotIncluded(str) && elf.IsElfEntry(str) && IsHaveDlopen(str)
	})

	if len(elf_have_entry_list) == 0 {
		log.Logger.Warnf("have not search include entry elf file with:", ts.SearchPath)
		return false
	}

	ts.ElfEntrySoPath = make(map[string]uint)
	for _, v := range elf_have_entry_list {
		log.Logger.Debugf("process path: %s", v)
		if ret, err := GetDlopenDepends(v); err != nil {
			continue
		} else {
			log.Logger.Debugf("%v", ret)
			if len(ret) == 0 {
				continue
			} else {
				entry_dlopen_so := Filter(ret, func(str string) bool {
					return !IsNotIncluded(str) && len(str) < 255
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
DLOPEN_SOPATH=$(cat /tmp/libcache.db | grep "{{ $element }}" | awk '{print $4}'|head -n 1)
[[ -f ${DLOPEN_SOPATH} ]] && (echo ${DLOPEN_SOPATH} >> /tmp/elfsonamelist)
[[ -f ${DLOPEN_SOPATH} ]] && (ldd ${DLOPEN_SOPATH} | awk '{print $3}' | sort| uniq | sed '/^$/d' >> /tmp/elfsonamelist)
{{end}}
{{ if len .ELFNameString }}
ldd {{.ELFNameString}} | awk '{print $3}' | sort| uniq | sed '/^$/d' >> /tmp/elfsonamelist
{{end}}

{{ if len .OutputNameString}}
echo > {{.OutputNameString}}
{{end}}

[[ -f /tmp/elfsonamelist ]] && (cat /tmp/elfsonamelist | sort | uniq | sed '/^$/d' >>  {{.OutputNameString}})

rm -v /tmp/libcache.db
rm -v /tmp/elfsonamelist

echo elfldd
`

/*!
 * @brief RenderElfWithLDD
 * @param output output file
 * @param save save file
 * @return bool, error
 */
func (ts *BinFormatReactor) RenderElfWithLDD(output, save string) (bool, error) {

	// init template
	log.Logger.Debug("render elf with ldd : ", ts.SearchPath)
	tpl, err := template.New("elfldd").Parse(TMPL_ELF_LDD)

	if err != nil {
		log.Logger.Fatalf("parse deb shell template failed! ", err)
		return false, nil
	}

	elfLDDShell := ElfLDDShellTemplate{"", make([]string, 0), output, comm.ConfigInfo.Verbose}

	for elfStr := range ts.ElfLDDPath {
		elfLDDShell.ELFNameString += elfStr
		elfLDDShell.ELFNameString += " "
	}

	for elfStr := range ts.ElfEntrySoPath {
		elfLDDShell.DlopenNameString = append(elfLDDShell.DlopenNameString, elfStr)
	}

	// create save file
	log.Logger.Debug("create save file: ", save)
	saveFd, ret := os.Create(save)
	if ret != nil {
		log.Logger.Fatalf("save to %s failed!", save)
		return false, nil
	}
	defer saveFd.Close()

	// render template
	// log.Logger.Debug("render template: ", elfLDDShell)
	tpl.Execute(saveFd, elfLDDShell)

	return true, nil
}

func GetDlopenDepends(path string) ([]string, error) {
	// strings /bin/bash | grep  "\.so"
	cmd := fmt.Sprintf("strings %s | egrep '^\\S+\\.so[.0-9]*$'", path)
	if msg, ret, err := comm.ExecAndWait(10, "bash", "-c", cmd); err != nil {
		comm.LoggerVerbose("check elf entry failed: %v", err, msg, ret)
		return nil, err
	} else {
		return strings.Split(msg, "\n"), nil
	}

}

func GetElfNeedWithLDD(elfSearchDir string) (string, error) {
	log.Logger.Debug("get elf need with ldd: ", elfSearchDir)
	return "", nil
}

func GetFindElfMissDepends(elfSearchDir string) (bool, error, []string) {

	// find . -type f  -exec file {} \; | grep  ELF | awk -F: '{print $1}' | xargs -I{} ldd {} | grep -i "not found"
	log.Logger.Debug("get find elf miss depends: ", elfSearchDir)
	elf_binary_path, err := elf.GetElfWithPath(elfSearchDir)
	if err != nil {
		log.Logger.Debugf("get elf with path failed! %s", err)
	}
	log.Logger.Debug("elf binary path: ", elf_binary_path)
	// fixme:(heysion) fix get elf binary path with depend list
	return false, nil, nil

}

func GetElfNeedWithStrace(elf string) (string, error) {
	log.Logger.Debug("get elf need with strace: ", elf)
	return "", nil
}

func ChrootExecShell(chrootDirPath, shell string, bindMounts []string) (bool, string, error) {
	log.Logger.Debugf("chroot exec shell: %s shell: %s", chrootDirPath, shell)

	// fixme:(heysion) mount /mnt/workdir/debdir/ to chroot /mnt/workdir/debdir
	if len(bindMounts) > 0 {
		for _, srcPath := range bindMounts {
			dstPath := chrootDirPath + srcPath
			fs.CreateDir(dstPath)
			log.Logger.Debug("bind mount: ", srcPath, dstPath)
			// bind mount src to dst
			if _, msg, err := comm.ExecAndWait(10, "mount", "-B", srcPath, dstPath); err != nil {
				log.Logger.Fatalf("mount %s to %s failed! ", srcPath, dstPath, err, msg)
			}
			// defer func() { RemovePath(dstPath) }()
			defer func() { log.Logger.Debugf("Umount %s", dstPath) }()
			defer func() { comm.UmountPath(dstPath) }()
			defer func() { log.Logger.Debugf("umount %s", dstPath) }()
		}

	}

	// mount shell to chroot
	shellSrcPath := fs.GetFilePPath(shell)
	shellDstPath := chrootDirPath + shellSrcPath
	shellChrootPath := chrootDirPath + shell

	log.Logger.Debugf("shell src path: %s to %s", shellSrcPath, shellDstPath)
	if ret, err := fs.CheckFileExits(shellDstPath); err != nil && !ret {
		fs.CreateDir(shellDstPath)
	}

	if _, msg, err := comm.ExecAndWait(10, "mount", "-B", shellSrcPath, shellDstPath); err != nil {
		log.Logger.Fatalf("mount %s to %s failed! ", shell, shellDstPath, err, msg)
		return false, msg, err
	}

	// fs.CreateDir(shellDstPath)
	// defer func() { RemovePath(shellDstPath) }()
	defer func() { log.Logger.Debugf("remove %s", shellDstPath) }()

	defer func() { comm.UmountPath(shellDstPath) }()
	defer func() { log.Logger.Debugf("umount %s", shellDstPath) }()

	// chmod +x shell
	if _, msg, err := comm.ExecAndWait(10, "chmod", "+x", "-R", shellChrootPath); err != nil {
		log.Logger.Fatalf("chmod +x %s failed! ", shellChrootPath, err, msg)
		return false, msg, err
	}

	// chroot shell
	log.Logger.Debugf("chroot shell: path: %s shell:%s", chrootDirPath, shell)
	if ret, msg, err := comm.ExecAndWait(4096, "chroot", chrootDirPath, shell); err != nil {
		log.Logger.Fatalf("chroot exec shell failed! ", err, msg, ret)
		return false, msg + ret, err
	} else {
		log.Logger.Info("chroot %s end.", shell)
		return true, ret, nil
	}
}

func ChrootExecShellBare(chroot string, shell string) (bool, string, error) {
	// chmod +x shell
	if _, msg, err := comm.ExecAndWait(10, "chmod", "+x", "-R", shell); err != nil {
		log.Logger.Fatalf("chmod +x %s failed! ", shell, err, msg)
		return false, msg, err
	}

	// chroot shell
	if strings.HasPrefix(shell, chroot) {
		shell = strings.Replace(shell, chroot, "", 1)
	}
	log.Logger.Debugf("chroot shell: path: %s shell:%s", chroot, shell)
	if ret, msg, err := comm.ExecAndWait(4096, "chroot", chroot, shell); err != nil {
		log.Logger.Fatalf("chroot exec shell failed! ", err, msg, ret)
		return false, msg, err
	} else {
		log.Logger.Debugf("chroot exec shell msg:", ret, msg)
	}
	return true, "", nil
}

type DebShellTemplate struct {
	ExtraPackageStr string
	DebString       string
	PreCommand      string
	PostCommand     string
	Verbose         bool
}

const DEB_SHELL_TMPL = `#!/bin/bash
{{if .Verbose }}set -x {{end}}
function apt_install_pkgs {
    DEBIAN_FRONTEND=noninteractive apt-get install -y {{.ExtraPackageStr}}
    DEBIAN_FRONTEND=noninteractive apt-get install -f -y
    echo apt_install_pkgs
}

function apt_update {
    DEBIAN_FRONTEND=noninteractive apt-get update
}

function apt_install_deb {
    DEBIAN_FRONTEND=noninteractive apt-get install -y {{.DebString}}
    DEBIAN_FRONTEND=noninteractive apt-get install -f -y
    echo apt_install_deb
}

function pre_command {
    {{if len .PreCommand }}{{.PreCommand}}{{end}}
    echo pre_command
}

function post_command {
    {{if len .PostCommand }}{{.PostCommand}}{{end}}
    echo post_command
}

{{if len .PreCommand }}pre_command{{end}}
apt_update
{{if len .DebString }}apt_install_deb {{end}}
{{if len .ExtraPackageStr }}apt_install_pkgs{{end}}
{{if len .PostCommand }}post_command{{end}}
`

func RenderDebConfig(DebConf comm.DebConfig, save string) (bool, error) {

	// init template
	// log.Logger.Debug("render deb config: ", DebConf)
	tpl, err := template.New("pica").Parse(DEB_SHELL_TMPL)

	if err != nil {
		log.Logger.Fatalf("parse deb shell template failed! ", err)
		return false, nil
	}

	debShell := DebShellTemplate{"", "", "", "", comm.ConfigInfo.Verbose}

	for _, debStr := range DebConf.FileElement.Deb {

		// log.Logger.Debugf("deb str: %s path :%s", debStr, debStr.Path)
		debShell.DebString += debStr.Path
		debShell.DebString += " "
	}

	if len(DebConf.FileElement.Package) != 0 {
		debShell.ExtraPackageStr = strings.Join(DebConf.FileElement.Package, " ")
	} else {
		debShell.ExtraPackageStr = ""
	}

	log.Logger.Debugf("chroot info command: %+v", DebConf.ChrootInfo)
	// PreCommand
	if len(DebConf.ChrootInfo.PreCmd) > 0 {

		debShell.PreCommand = DebConf.ChrootInfo.PreCmd

	}
	// PostCommand
	if len(DebConf.ChrootInfo.PostCmd) > 0 {

		debShell.PostCommand = DebConf.ChrootInfo.PostCmd

	}

	// create save file
	log.Logger.Debug("create save file: ", save)
	saveFd, ret := os.Create(save)
	if ret != nil {
		log.Logger.Fatalf("save to %s failed!", save)
		return false, nil
	}
	defer saveFd.Close()

	// render template
	log.Logger.Debug("render template: ", debShell)
	tpl.Execute(saveFd, debShell)

	return true, nil
}
