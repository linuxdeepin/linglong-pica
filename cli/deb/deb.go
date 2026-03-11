/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package deb

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"pault.ag/go/debian/control"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/cli/linglong"
	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

var (
	versionConstraintPattern = regexp.MustCompile(`\([^)]*\)`)
	archQualifierPattern     = regexp.MustCompile(`\[[^]]*\]`)
	profileQualifierPattern  = regexp.MustCompile(`<[^>]*>`)
)

type Deb struct {
	Name           string
	Id             string
	Type           string
	Ref            string
	Hash           string
	Path           string
	Package        string `control:"Package"`
	Version        string `control:"Version"`
	SHA256         string `control:"SHA256"`
	Desc           string `control:"Description"`
	PreDepends     string `control:"Pre-Depends"`
	Depends        string `control:"Depends"`
	Architecture   string `control:"Architecture"`
	Filename       string `control:"Filename"`
	SourcePackage  string
	FromAppStore   bool
	PackageKind    string
	Command        []string
	RuntimeDepends []string
	BuildDepends   []string
	Build          []string
	DelMap         map[string]bool // 用来记录跳过的包的映射，每个Deb实例独立
}

func (d *Deb) GetPackageUrl(source, distro, arch string) string {
	_ = source
	_ = distro
	_ = arch
	return AptDownload(d.Name)
}

func (d *Deb) CheckDebHash() bool {
	info, err := os.Stat(d.Path)
	if err != nil {
		log.Logger.Warn(err)
		return false
	}
	if info.Size() == 0 {
		log.Logger.Warnf("%s is empty", d.Path)
		return false
	}

	hash, err := fs.GetFileSha256(d.Path)
	if d.Hash == "" {
		log.Logger.Debugf("%s not verify hash", d.Name)
		d.Hash = hash
		return true
	}
	if err != nil {
		log.Logger.Warn(err)
		d.Hash = hash
		return false
	}
	return hash == d.Hash
}

// FetchDebFile
func (d *Deb) FetchDebFile(dstPath string) bool {
	log.Logger.Debugf("FetchDebFile %s,ts:%v type:%s", dstPath, d, d.Type)

	if d.Type == "repo" {
		fs.CreateDir(fs.GetFilePPath(dstPath))
		downloadDir := fs.GetFilePPath(dstPath)
		debFileName := filepath.Base(dstPath)
		existingDebPath := filepath.Join(downloadDir, debFileName)
		if err := os.Remove(existingDebPath); err != nil && !os.IsNotExist(err) {
			log.Logger.Warnf("remove cached deb failed: %v", err)
			return false
		}
		if ret, msg, err := comm.ExecAndWaitInDir(1<<20, downloadDir, "apt", "download", d.Name, "-y"); err != nil {
			log.Logger.Warnf("msg: %+v, out: %+v", msg, err, ret)
			return false
		} else {
			log.Logger.Debugf("ret: %+v", ret)
		}

		if ret, err := fs.CheckFileExits(dstPath); ret {
			d.Path = dstPath
			return true
		} else {
			log.Logger.Warnf("downalod %s , err:%+v", dstPath, err)
			return false
		}
	} else if d.Type == "local" {
		if ret, err := fs.CheckFileExits(d.Ref); !ret {
			log.Logger.Warnf("not exist ! %s , err:%+v", d.Ref, err)
			return false
		}

		fs.CreateDir(fs.GetFilePPath(dstPath))
		if ret, msg, err := comm.ExecAndWait(1<<8, "cp", "-v", d.Ref, dstPath); err != nil {
			log.Logger.Fatalf("msg: %+v err:%+v, out: %+v", msg, err, ret)
			return false
		} else {
			log.Logger.Debugf("ret: %+v", ret)
		}

		if ret, err := fs.CheckFileExits(dstPath); ret {
			d.Path = dstPath
			return true
		} else {
			log.Logger.Warnf("downalod %s , err:%+v", dstPath, err)
			return false
		}
	}
	return false
}

func (d *Deb) ExtractDeb() error {
	if ret, err := AptShow(d.Path); err != nil {
		return err
	} else {
		// apt-cache show Unmarshal
		info, err := control.ParseControl(bufio.NewReader(strings.NewReader(ret)), "")
		if err != nil {
			log.Logger.Warnf("parse control error: %s", err)
			return err
		}
		d.Package = info.Source.Paragraph.Values["Package"]
		// 格式化成玲珑使用的四位版本号，剔除非数字部分
		d.Version = formatVersion(info.Source.Paragraph.Values["Version"])
		d.SHA256 = info.Source.Paragraph.Values["SHA256"]
		// 在描述信息里添加原包的版本号信息
		d.Desc = fmt.Sprintf("convert from %s    %s", info.Source.Paragraph.Values["Version"], strings.ReplaceAll(info.Source.Paragraph.Values["Description"], "\n", ""))
		d.PreDepends = info.Source.Paragraph.Values["Pre-Depends"]
		d.Depends = info.Source.Paragraph.Values["Depends"]
		d.SourcePackage = cleanSourcePackage(info.Source.Paragraph.Values["Source"])
		if d.SourcePackage == "" {
			d.SourcePackage = d.Package
		}
		if info.Source.Paragraph.Values["Architecture"] == "all" {
			d.Architecture = runtime.GOARCH
		} else {
			d.Architecture = info.Source.Paragraph.Values["Architecture"]
		}
		d.Filename = info.Source.Paragraph.Values["Filename"]
	}

	// 解压 deb 包，部分内容需要从解开的包中获取
	debDirPath := filepath.Join(filepath.Dir(d.Path), d.Name)
	if ret, msg, err := comm.ExecAndWait(1<<20, "dpkg-deb", "-x", d.Path, debDirPath); err != nil {
		log.Logger.Warnf("msg: %+v err:%+v, out: %+v", msg, err, ret)
		return err
	} else {
		log.Logger.Debugf("ret: %+v", ret)
		// 应用商店的 deb 包，包含 opt/apps 目录，针对该目录是否存在，判定是否为应用商店包
		targetPath := filepath.Join(debDirPath, "opt/apps")
		if ret, _ := fs.CheckFileExits(targetPath); ret {
			log.Logger.Infof("%s is from app-store", d.Name)
			d.FromAppStore = true
		} else {
			log.Logger.Infof("%s is not from app-store", d.Name)
		}
	}

	// 应用需要指定，四位版本号
	parts := strings.Split(d.Version, ".")
	numParts := len(parts)
	// 补足缺失的部分，使其成为四位版本号
	for numParts < 4 {
		parts = append(parts, "0")
		numParts++
	}
	d.Version = strings.Join(parts[:4], ".") // 只取前四个部分组成新的版本号

	return nil
}

// 解析依赖
func (d *Deb) ResolveDepends() {
	d.RuntimeDepends = parseDependencyNames([]string{d.PreDepends, d.Depends}, buildRuntimeExcludeSet())
	d.BuildDepends = d.resolveBuildDepends()
}

func (d *Deb) GenerateBuildScript() {
	d.Build = nil
	d.Build = append(d.Build, []string{
		"#>>> auto generate by ll-pica begin",
		"set -x",
	}...)

	d.Build = append(d.Build, []string{
		"# set the local sources directory",
		fmt.Sprintf("EXTERNAL_DEB_SOURCES=\"%s\"", comm.LlLocalSourceDir),
	}...)

	d.PackageKind = "app"
	debDirPath := filepath.Join(filepath.Dir(d.Path), d.Name)

	// 如果是应用商店的软件包
	if d.FromAppStore {
		// 删除多余的 desktop 文件
		if ret, msg, err := comm.ExecAndWait(10, "sh", "-c",
			fmt.Sprintf("find %s -name '*.desktop' | grep _uos | xargs -I {} rm {}", debDirPath)); err != nil {
			log.Logger.Warnf("remove extra desktop file error: %+v", msg)
		} else {
			log.Logger.Debugf("remove extra desktop file: %+v", ret)
		}
	}

	desktopFiles, msg, err := comm.ExecAndWait(10, "sh", "-c", fmt.Sprintf("find %s -name '*.desktop' | grep applications", debDirPath))
	if err != nil {
		log.Logger.Fatalf("find desktop error: %s out: %s", msg, desktopFiles)
	}

	// 读取desktop 文件
	var desktopData fs.DesktopData
	var status bool
	var execLine, iconValue, realPackageName string

	// 商店包存在包名和 内部 appid 名无法对应的情况，获取解包后真实路径
	if entries, err := os.ReadDir(debDirPath + "/opt/apps"); err == nil {
		realPackageName = entries[0].Name()
	}

	// 如果存在多个 desktop 文件进行循环, 生成对应的 sed 操作
	for _, desktop := range strings.Split(desktopFiles, "\n") {
		if desktop == "" {
			continue
		}
		status, desktopData = fs.DesktopInit(desktop)
		if !status {
			log.Logger.Errorf("load desktop error: %s", desktop)
			continue
		}

		//获取 desktop 文件，Exec 行的内容,并且对字符串做处理
		pattern := regexp.MustCompile(`Exec=|"|\n`)
		execLine = pattern.ReplaceAllLiteralString(desktopData["Desktop Entry"]["Exec"], "")

		// 正则表达式，匹配"/usr/"或者"/opt/apps/$appid/file/"，参考 https://regex101.com/r/oyo0YX/1
		pattern = regexp.MustCompile(`/usr/|/opt/apps/[^/]+/files/`)

		// 使用正则表达式找到匹配的部分并替换
		execLine = pattern.ReplaceAllLiteralString(execLine, fmt.Sprintf("/opt/apps/%s/files/", d.Id))
		execLine = normalizeExecLine(execLine, debDirPath, d.Id)

		iconValue = fs.TransIconToLl(desktopData["Desktop Entry"]["Icon"])
		index := strings.Index(desktop, comm.LlLocalSourceDir)
		if index != -1 {
			// 如果找到了子串，则移除它及其之前的部分
			modiDesktopPath := "$EXTERNAL_DEB_SOURCES" + desktop[index+len(comm.LlLocalSourceDir):]
			d.Build = append(d.Build, []string{
				"# modify desktop, Exec and Icon should not contanin absolut paths",
				fmt.Sprintf("sed -i '/Exec*/c\\Exec=%s' %s", execLine, modiDesktopPath),
				fmt.Sprintf("sed -i '/Icon*/c\\Icon=%s' %s", iconValue, modiDesktopPath),
			}...)
		}
	}

	// 以 sh 后缀的脚本，替换脚本中的路径，考虑到deb包定义包名和玲珑id名不一样的情况
	if execFiles, _, err := comm.ExecAndWait(10, "sh", "-c",
		// 找到可执行文件，以 .sh 后缀的脚本。统一将内部的原包名替换为设置的玲珑id，少部分存在没写 .sh 后缀的人工判断，其他方式很难判断该脚本为shell.
		fmt.Sprintf("find %s -name '*.sh'", debDirPath)); err == nil {
		for _, execFile := range strings.Split(execFiles, "\n") {
			if execFile == "" {
				continue
			}
			index := strings.Index(execFile, comm.LlLocalSourceDir)
			if index != -1 {
				// 如果找到了子串，则移除它及其之前的部分
				modiExecFilePath := "$EXTERNAL_DEB_SOURCES" + execFile[index+len(comm.LlLocalSourceDir):]
				d.Build = append(d.Build, []string{
					fmt.Sprintf("sed -i 's/%s/%s/g' %s", d.Name, d.Id, modiExecFilePath),
				}...)
			}
		}
	}

	d.Build = append(d.Build, []string{
		"",
		"install -d $PREFIX/share",
		"install -d $PREFIX/bin",
		"install -d $PREFIX/lib",
	}...)

	if d.FromAppStore {
		d.Build = append(d.Build, []string{
			"",
			"# move files",
			fmt.Sprintf("cp -r $EXTERNAL_DEB_SOURCES/%s/opt/apps/%s/entries/* $PREFIX/share", d.Name, realPackageName),
			fmt.Sprintf("cp -rf $EXTERNAL_DEB_SOURCES/%s/opt/apps/%s/files/* $PREFIX", d.Name, realPackageName),
		}...)
	}

	if _, err := os.ReadDir(debDirPath + "/usr"); err == nil {
		d.Build = append(d.Build, []string{
			"",
			"# move files",
			fmt.Sprintf("cp -r $EXTERNAL_DEB_SOURCES/%s/usr/* $PREFIX", d.Name),
		}...)
	}

	d.Build = append(d.Build, "#>>> auto generate by ll-pica end")

	d.Command = strings.Split(execLine, " ")
}

func (d *Deb) resolveBuildDepends() []string {
	if d.SourcePackage == "" {
		return nil
	}

	ret, err := AptShowSource(d.SourcePackage)
	if err != nil {
		log.Logger.Warnf("apt-cache showsrc %s failed: %v", d.SourcePackage, err)
		return nil
	}

	info, err := control.ParseControl(bufio.NewReader(strings.NewReader(ret)), "")
	if err != nil {
		log.Logger.Warnf("parse source control error for %s: %v", d.SourcePackage, err)
		return nil
	}

	return parseDependencyNames([]string{
		info.Source.Paragraph.Values["Build-Depends"],
		info.Source.Paragraph.Values["Build-Depends-Indep"],
	}, nil)
}

func buildRuntimeExcludeSet() map[string]struct{} {
	exclude := make(map[string]struct{})
	for _, pkg := range []string{"deepin-elf-verify", "systemd", "systemd-dev", "usrmerge", "xdg-utils", "dbus", "dbus-broker"} {
		exclude[pkg] = struct{}{}
	}

	cli := linglong.NewLinglongCli()
	for _, pkg := range cli.GetBaseInsPack() {
		exclude[pkg] = struct{}{}
	}
	for _, pkg := range cli.GetRuntimeInsPack() {
		exclude[pkg] = struct{}{}
	}
	return exclude
}

func parseDependencyNames(fields []string, exclude map[string]struct{}) []string {
	var result []string
	seen := make(map[string]struct{})
	for _, field := range fields {
		for _, item := range strings.Split(field, ",") {
			for _, candidate := range strings.Split(item, "|") {
				name := normalizeDependencyName(candidate)
				if name == "" || isDebhelperPackage(name) {
					continue
				}
				if _, skipped := exclude[name]; skipped {
					continue
				}
				if _, ok := seen[name]; ok {
					continue
				}
				seen[name] = struct{}{}
				result = append(result, name)
			}
		}
	}
	return result
}

func normalizeDependencyName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	value = versionConstraintPattern.ReplaceAllString(value, "")
	value = archQualifierPattern.ReplaceAllString(value, "")
	value = profileQualifierPattern.ReplaceAllString(value, "")
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	name := fields[0]
	if idx := strings.Index(name, ":"); idx != -1 {
		name = name[:idx]
	}
	if strings.HasPrefix(name, "${") {
		return ""
	}
	return strings.TrimSpace(name)
}

func cleanSourcePackage(value string) string {
	name := normalizeDependencyName(value)
	if name != "" {
		return name
	}
	value = strings.TrimSpace(value)
	if idx := strings.Index(value, "("); idx != -1 {
		value = value[:idx]
	}
	return strings.TrimSpace(value)
}

func isDebhelperPackage(name string) bool {
	return name == "debhelper" || strings.HasPrefix(name, "debhelper-")
}

func normalizeExecLine(execLine, debDirPath, packageID string) string {
	fields := strings.Fields(execLine)
	if len(fields) == 0 {
		return execLine
	}
	if strings.Contains(fields[0], "/") {
		return execLine
	}

	subpath := filepath.Join("usr", "bin", fields[0])
	if ret, _ := fs.CheckFileExits(filepath.Join(debDirPath, subpath)); ret {
		fields[0] = filepath.ToSlash(filepath.Join("/opt/apps", packageID, "files", "bin", fields[0]))
		return strings.Join(fields, " ")
	}
	return execLine
}

// 将从包里获取的版本号格式化成四位数
func formatVersion(versionStr string) string {
	// 先尝试直接按点分割，处理常规的版本号格式
	parts := strings.Split(versionStr, ".")

	var digits []string
	for _, part := range parts {
		// 将字符串转换成数字，如果没有报错，说明是纯数字
		if _, err := strconv.Atoi(part); err == nil {
			// 大于 1 位数字，去除前导零
			if len(part) > 1 {
				digits = append(digits, strings.TrimLeft(part, "0"))
			} else {
				digits = append(digits, part)
			}
		} else {
			// 查找并提取非数字部分后的数字
			re := regexp.MustCompile(`\d+`)
			match := re.FindString(part)
			if match != "" {
				digits = append(digits, strings.TrimLeft(match, "0"))
			}
		}
	}

	// 确保版本号至少有四个段，不足则用0填充
	for len(digits) < 4 {
		digits = append(digits, "0")
	}

	// 截取前四个有效数字段进行格式化
	formattedVersion := strings.Join(digits[:4], ".")
	return formattedVersion
}
