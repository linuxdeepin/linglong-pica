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

	// copy shell to chroot
	// fixme: recommand to used io.Copy()
	// if _, msg, err := ExecAndWait(10, "cp", "-v", shell, shellChroot); err != nil {
	// 	logger.Fatalf("copy %s to %s failed! ", shell, shellChroot, err, msg)
	// }
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

func RenderDebConfig(debConf DebConfig, save string) (bool, error) {

	// init template
	logger.Debug("render deb config: ", debConf)
	tpl, err := template.New("pica").Parse(DEB_SHELL_TMPL)

	if err != nil {
		logger.Fatalf("parse deb shell template failed! ", err)
		return false, nil
	}

	debShell := DebShellTemplate{"", "", false}

	for _, debStr := range debConf.FileElement.Deb {

		logger.Debugf("deb str: %s path :%s", debStr, debStr.Path)
		debShell.DebString += debStr.Path
		debShell.DebString += " "
	}

	if len(debConf.FileElement.Package) != 0 {
		debShell.ExtraPackageStr = strings.Join(debConf.FileElement.Package, " ")
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
