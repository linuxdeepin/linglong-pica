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
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/smira/flag"
	"pault.ag/go/debian/control"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/cmd"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/query"
	"github.com/aptly-dev/aptly/utils"

	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/cli/linglong"
	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

type Deb struct {
	Name         string
	Id           string
	Type         string
	Ref          string
	Hash         string
	Path         string
	Package      string `control:"Package"`
	Version      string `control:"Version"`
	SHA256       string `control:"SHA256"`
	Desc         string `control:"Description"`
	Depends      string `control:"Depends"`
	Architecture string `control:"Architecture"`
	Filename     string `control:"Filename"`
	FromAppStore bool
	PackageKind  string
	Command      []string
	Sources      []comm.Source
	Build        []string
	DelMap       map[string]bool // 用来记录跳过的包的映射，每个Deb实例独立
}

func (d *Deb) GetPackageUrl(source, distro, arch string) string {
	aptlyCache := comm.AptlyCachePath()
	// 删除掉aptly缓存的内容
	if ret, _ := fs.CheckFileExits(aptlyCache); ret {
		log.Logger.Debugf("%s is existd!", aptlyCache)
		if ret, err := fs.RemovePath(aptlyCache); err != nil {
			log.Logger.Warnf("err:%+v, out: %+v", err, ret)
		}
	}

	root := cmd.RootCommand()
	root.UsageLine = "aptly"

	// 只过滤需要搜索的包
	args := []string{
		"mirror",
		"create",
		"-ignore-signatures",
		"-architectures=" + arch,
		"-filter=" + d.Name,
		distro,
		source,
		distro,
	}

	cmd.Run(root, args, cmd.GetContext() == nil)

	d.GetPackageList(distro)
	if len(d.Sources) > 0 {
		return d.Sources[0].Url
	} else {
		log.Logger.Warnf("%s not found url, fallback to apt download", d.Name)
		return AptDownload(d.Name)
	}
}

func (d *Deb) CheckDebHash() bool {
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

		if ret, msg, err := comm.ExecAndWait(1<<20, "wget", "-O", dstPath, d.Ref); err != nil {
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
		d.Depends = info.Source.Paragraph.Values["Depends"]
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

	if d.Type != "local" {
		d.Sources = append(d.Sources, comm.Source{Kind: "file", Digest: d.Hash, Url: d.Ref})
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
func (d *Deb) ResolveDepends(source, distro string, withDep bool) {
	// 可能存在依赖为空的情况
	if d.Depends == "" {
		return
	}

	// mirror_update_args []string
	// 玲珑作为单应用程序，不需要在意里面的版本冲突，直接选择最新版本
	// 定义一个正则表达式，删除匹配括号及其中的内容
	reParentheses := regexp.MustCompile(`\([^)]*\)`)
	filter := strings.Replace(reParentheses.ReplaceAllString(d.Depends, ""), ",", "|", -1)
	// 移除所有空格
	reSpace := regexp.MustCompile(`\s+`)
	filter = reSpace.ReplaceAllString(filter, "")

	// 设置黑名单过滤包，不获取依赖
	skipPackage := []string{"deepin-elf-verify", "systemd", "systemd-dev", "usrmerge", "xdg-utils", "dbus", "dbus-broker"}
	cli := linglong.NewLinglongCli()
	// 过滤掉 base 中安装过的包
	skipPackage = append(skipPackage, cli.GetBaseInsPack()...)
	// 过滤掉 runtime 中安装过的包
	skipPackage = append(skipPackage, cli.GetRuntimeInsPack()...)
	filterSlice := strings.Split(filter, ",")
	d.DelMap = make(map[string]bool) // 初始化为每个Deb独立的map
	for _, item := range skipPackage {
		d.DelMap[item] = true
	}
	var result []string
	for _, item := range filterSlice {
		if !d.DelMap[item] {
			result = append(result, item)
		}
	}
	filter = strings.Join(result, ",")

	// 删除掉aptly缓存的内容
	aptlyCache := comm.AptlyCachePath()
	if ret, _ := fs.CheckFileExits(aptlyCache); ret {
		log.Logger.Debugf("%s is existd!", aptlyCache)
		if ret, err := fs.RemovePath(aptlyCache); err != nil {
			log.Logger.Warnf("err:%+v, out: %+v", err, ret)
		}
	}

	if d.Architecture == "" || d.Name == "" {
		log.Logger.Errorf("arch or package name is empty")
		return
	}

	// 依赖为空，不需要处理
	if filter == "" {
		return
	}

	root := cmd.RootCommand()
	root.UsageLine = "aptly"

	args := []string{
		"mirror",
		"create",
		"-ignore-signatures",
		"-architectures=" + d.Architecture,
		"-filter=" + filter,
	}

	if withDep {
		args = append(args, "-filter-with-deps")
	}

	args = append(args, []string{
		distro,
		source,
		distro,
	}...)

	cmd.Run(root, args, cmd.GetContext() == nil)

	d.GetPackageList(distro)
}

func (d *Deb) GenerateBuildScript() {
	d.Build = append(d.Build, []string{
		"#>>> auto generate by ll-pica begin",
		"set -x",
	}...)

	// 设置 linglong/sources 目录
	d.Build = append(d.Build, []string{
		"# set the linglong/sources directory",
		fmt.Sprintf("SOURCES=\"%s\"", comm.LlSourceDir),
		fmt.Sprintf("dpkg-deb -x $SOURCES/%s $SOURCES/%s", filepath.Base(d.Filename), d.Name),
	}...)

	d.PackageKind = "app"
	// linglong/sources 下解压 app 后的目录
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

		iconValue = fs.TransIconToLl(desktopData["Desktop Entry"]["Icon"])
		index := strings.Index(desktop, comm.LlSourceDir)
		if index != -1 {
			// 如果找到了子串，则移除它及其之前的部分
			modiDesktopPath := "$SOURCES" + desktop[index+len(comm.LlSourceDir):]
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
			index := strings.Index(execFile, comm.LlSourceDir)
			if index != -1 {
				// 如果找到了子串，则移除它及其之前的部分
				modiExecFilePath := "$SOURCES" + execFile[index+len(comm.LlSourceDir):]
				d.Build = append(d.Build, []string{
					fmt.Sprintf("sed -i 's/%s/%s/g' %s", d.Name, d.Id, modiExecFilePath),
				}...)
			}
		}
	}

	// 玲珑内部的 /opt/apps 路径拼接的是 linglong-id
	d.Build = append(d.Build, []string{
		"OUT_DIR=\"$(mktemp -d)\"", // 临时目录，处理完内容再移动到$PREFIX
		"DEPS_LIST=\"$OUT_DIR/DEPS.list\"",
		"find $SOURCES -type f -name \"*.deb\" > $DEPS_LIST",
		"DATA_LIST_DIR=\"$OUT_DIR/data\"", // 包数据存放的临时目录
		"mkdir -p /tmp/deb-source-file",   // 用于记录安装的所有文件来自哪个包
		"while IFS= read -r file",
		"do",
		"    CONTROL_FILE=$(ar -t $file | grep control.tar)", // 提取control文件
		"    ar -x \"$file\" $CONTROL_FILE",
		"    PKG=$(tar -xf $CONTROL_FILE ./control -O | grep '^Package:' | awk '{print $2}')", // 获取包名
		"    rm $CONTROL_FILE || true",
		"    DATA_FILE=$(ar -t $file | grep data.tar)", // 提取data.tar文件
		"    ar -x $file $DATA_FILE",
		"    mkdir -p $DATA_LIST_DIR",
		"    tar -xvf $DATA_FILE -C $DATA_LIST_DIR >> \"/tmp/deb-source-file/$(basename $file).list\"", // 解压data.tar文件到输出目录
		"    rm -rf $DATA_FILE 2>/dev/null || true",
		"    rm -r ${DATA_LIST_DIR:?}/usr/share/applications* 2>/dev/null || true",                           // 清理不需要复制的目录
		"    sed -i \"s#/usr#$PREFIX#g\" $DATA_LIST_DIR/usr/lib/$TRIPLET/pkgconfig/*.pc 2>/dev/null || true", // # 修改pc文件的prefix
		"    sed -i \"s#/usr#$PREFIX#g\" $DATA_LIST_DIR/usr/share/pkgconfig/*.pc 2>/dev/null || true",
		"    find $DATA_LIST_DIR -type l | while IFS= read -r file; do", // 修改指向/lib的绝对路径的软链接
		"        Link_Target=$(readlink $file)",
		"        if echo $Link_Target | grep -q ^/lib && ! [ -f $Link_Target ]; then", // 如果指向的路径以/lib开头，并且文件不存在，则添加 /runtime 前缀, 部分 dev 包会创建 so 文件的绝对链接指向 /lib 目录下
		"            ln -sf $PREFIX$Link_Target $file",
		"            echo \"    FIX LINK $Link_Target => $PREFIX$Link_Target\"",
		"        fi",
		"    done",
		"    find $DATA_LIST_DIR -type f -exec file {} \\; | grep 'shared object' | awk -F: '{print $1}' | while IFS= read -r file; do", // 修复动态库的RUNPATH
		"        runpath=$(readelf -d $file | grep RUNPATH |  awk '{print $NF}')",
		"        if echo $runpath | grep -q '^\\[/'; then", // 如果RUNPATH使用绝对路径，则添加/runtime前缀
		"            runpath=${runpath#[}",
		"            runpath=${runpath%]}",
		"            newRunpath=${runpath//usr\\/lib/runtime\\/lib}",
		"            newRunpath=${newRunpath//usr/runtime}",
		"            patchelf --set-rpath $newRunpath $file",
		"            echo \"    FIX RUNPATH $file $runpath => $newRunpath\"",
		"        fi",
		"    done",
		"    cp -rP $DATA_LIST_DIR/lib $PREFIX 2>/dev/null || true",
		"    cp -rP $DATA_LIST_DIR/bin $PREFIX 2>/dev/null || true",
		"    cp -rP $DATA_LIST_DIR/usr/* $PREFIX 2>/dev/null || true",
		"done < \"$DEPS_LIST\"",
		"rm -r $OUT_DIR || true", // # 清理临时目录
	}...)

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
			fmt.Sprintf("cp -r $SOURCES/%s/opt/apps/%s/entries/* $PREFIX/share", d.Name, realPackageName),
			fmt.Sprintf("cp -rf $SOURCES/%s/opt/apps/%s/files/* $PREFIX", d.Name, realPackageName),
		}...)
	}

	if _, err := os.ReadDir(debDirPath + "/usr"); err == nil {
		d.Build = append(d.Build, []string{
			"",
			"# move files",
			fmt.Sprintf("cp -r $SOURCES/%s/usr/* $PREFIX", d.Name),
		}...)
	}

	d.Build = append(d.Build, "#>>> auto generate by ll-pica end")

	d.Command = strings.Split(execLine, " ")
}

// 获取 deb 包
func (d *Deb) GetPackageList(distro string) {
	context := cmd.GetContext()
	defer context.Shutdown()
	collectionFactory := context.NewCollectionFactory()
	repo, err := collectionFactory.RemoteRepoCollection().ByName(distro)

	if err != nil {
		log.Logger.Errorf("unable to update: %s", err)
	}

	err = collectionFactory.RemoteRepoCollection().LoadComplete(repo)
	if err != nil {
		log.Logger.Errorf("unable to update: %s", err)
	}

	verifier, err := getVerifier(context.Flags())
	if err != nil {
		log.Logger.Errorf("unable to initialize GPG verifier: %s", err)
	}

	err = repo.Fetch(context.Downloader(), verifier)
	if err != nil {
		log.Logger.Errorf("unable to update: %s", err)
	}

	context.Progress().Printf("Downloading & parsing package files...\n")
	err = repo.DownloadPackageIndexes(context.Progress(), context.Downloader(), verifier, collectionFactory, false)
	if err != nil {
		log.Logger.Errorf("unable to update: %s", err)
	}

	if repo.Filter != "" {
		context.Progress().Printf("Applying filter...\n")
		var filterQuery deb.PackageQuery

		filterQuery, err = query.Parse(repo.Filter)
		if err != nil {
			log.Logger.Errorf("unable to update: %s", err)
		}

		var oldLen, newLen int
		oldLen, newLen, err = repo.ApplyFilter(context.DependencyOptions(), filterQuery, context.Progress())
		if err != nil {
			log.Logger.Errorf("unable to update: %s", err)
		}
		context.Progress().Printf("Packages filtered: %d -> %d.\n", oldLen, newLen)
	}

	var (
		downloadSize int64
		queue        []deb.PackageDownloadTask
	)

	context.Progress().Printf("Building download queue...\n")
	queue, downloadSize, err = repo.BuildDownloadQueue(context.PackagePool(), collectionFactory.PackageCollection(),
		collectionFactory.ChecksumCollection(nil), false)

	if err != nil {
		log.Logger.Errorf("unable to update: %s", err)
	}

	defer func() {
		// on any interruption, unlock the mirror
		err = context.ReOpenDatabase()
		if err == nil {
			repo.MarkAsIdle()
			collectionFactory.RemoteRepoCollection().Update(repo)
		}
	}()

	repo.MarkAsUpdating()
	err = collectionFactory.RemoteRepoCollection().Update(repo)
	if err != nil {
		log.Logger.Errorf("unable to update: %s", err)
	}

	err = context.CloseDatabase()
	if err != nil {
		log.Logger.Errorf("unable to update: %s", err)
	}

	context.GoContextHandleSignals()

	count := len(queue)
	context.Progress().Printf("Download queue: %d items (%s)\n", count, utils.HumanBytes(downloadSize))

	// Download from the queue
	context.Progress().InitBar(downloadSize, true, aptly.BarMirrorUpdateDownloadPackages)

	downloadQueue := make(chan int)

	var (
		errors  []string
		errLock sync.Mutex
	)

	pushError := func(err error) {
		errLock.Lock()
		errors = append(errors, err.Error())
		errLock.Unlock()
	}

	go func() {
		for idx := range queue {
			select {
			case downloadQueue <- idx:
			case <-context.Done():
				return
			}
		}
		close(downloadQueue)
	}()

	var wg sync.WaitGroup

	for i := 0; i < context.Config().DownloadConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case idx, ok := <-downloadQueue:
					if !ok {
						return
					}

					task := &queue[idx]

					var e error

					// 跳过黑名单
					if d.DelMap[strings.Split(task.File.Filename, "_")[0]] {
						continue
					}

					// 构造下载路径
					task.TempDownPath = filepath.Join(filepath.Dir(d.Path), task.File.Filename)
					// 下载文件
					e = context.Downloader().DownloadWithChecksum(
						context,
						repo.PackageURL(task.File.DownloadURL()).String(),
						task.TempDownPath,
						&task.File.Checksums,
						false)

					source := comm.Source{
						Kind:   "file",
						Url:    repo.PackageURL(task.File.DownloadURL()).String(),
						Digest: task.File.Checksums.SHA256,
					}
					// 返回 sources 列表，记录 kind, url, hash
					d.Sources = append(d.Sources, source)

					if e != nil {
						pushError(e)
						continue
					}

					task.Done = true

				case <-context.Done():
					return
				}
			}
		}()
	}

	// Wait for all download goroutines to finish
	wg.Wait()

	context.Progress().ShutdownBar()
}

func getVerifier(flags *flag.FlagSet) (pgp.Verifier, error) {
	context := cmd.GetContext()
	if cmd.LookupOption(context.Config().GpgDisableVerify, flags, "ignore-signatures") {
		return nil, nil
	}

	keyRings := flags.Lookup("keyring").Value.Get().([]string)

	verifier := context.GetVerifier()
	for _, keyRing := range keyRings {
		verifier.AddKeyring(keyRing)
	}

	err := verifier.InitKeyring()
	if err != nil {
		return nil, err
	}

	return verifier, nil
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
