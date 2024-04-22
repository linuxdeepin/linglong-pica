/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package deb

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/smira/flag"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/cmd"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/query"
	"github.com/aptly-dev/aptly/utils"
	"gopkg.in/yaml.v3"

	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

type Deb struct {
	Name    string
	Id      string
	Type    string
	Ref     string
	Hash    string
	Path    string
	Package string `yaml:"Package"`
	Version string `yaml:"Version"`
	SHA256  string `yaml:"SHA256"`
	// Desc         string `yaml:"Description"`
	Depends      string `yaml:"Depends"`
	Architecture string `yaml:"Architecture"`
	Filename     string `yaml:"Filename"`
	FromAppStore bool
	PackageKind  string
	Command      string
	Sources      []Source
	Build        []string
}

type Source struct {
	Kind   string
	Digest string
	Url    string
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
		d.Name,
		source,
		distro,
	}

	cmd.Run(root, args, cmd.GetContext() == nil)

	d.GetPackageList()
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
	if hash == d.Hash {
		return true
	}

	return true
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
		err = yaml.Unmarshal([]byte(ret), &d)
		if err != nil {
			log.Logger.Warnf("apt-cache show unmarshal error: %s", err)
			return err
		}
	}

	// 解压 deb 包，部分内容需要从解开的包中获取
	debDirPath := filepath.Join(filepath.Dir(d.Path), "app")
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
		d.Sources = append(d.Sources, Source{Kind: "file", Digest: d.Hash, Url: d.Ref})
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
func (d *Deb) ResolveDepends(source, distro string) {
	var (
		// mirror_update_args []string
		filter = strings.Replace(d.Depends, ",", "|", -1)
	)

	// 删除掉aptly缓存的内容
	aptlyCache := comm.AptlyCachePath()
	if ret, _ := fs.CheckFileExits(aptlyCache); ret {
		log.Logger.Debugf("%s is existd!", aptlyCache)
		if ret, err := fs.RemovePath(aptlyCache); err != nil {
			log.Logger.Warnf("err:%+v, out: %+v", err, ret)
		}
	}

	root := cmd.RootCommand()
	root.UsageLine = "aptly"

	if d.Architecture == "" || d.Name == "" || filter == "" {
		log.Logger.Fatal("arch or package name or filter is empty")
	}

	args := []string{
		"mirror",
		"create",
		"-ignore-signatures",
		"-architectures=" + d.Architecture,
		"-filter=" + filter,
		"-filter-with-deps",
		d.Name,
		source,
		distro,
	}

	cmd.Run(root, args, cmd.GetContext() == nil)

	d.GetPackageList()
}

// 对生成的 Source 数组进行去重
func (d *Deb) RemoveExcessDeps() {
	var result []Source
	uniqueMap := make(map[string]bool)
	for _, pkg := range d.Sources {
		key, _ := json.Marshal(pkg)
		// 如果 key 不存在于 map 中，则添加
		if _, ok := uniqueMap[string(key)]; !ok {
			uniqueMap[string(key)] = true
			result = append(result, pkg)
		}
	}
	d.Sources = result
}

func (d *Deb) GenerateBuildScript() {
	var archLinuxGnu string
	switch d.Architecture {
	case "amd64":
		archLinuxGnu = "x86_64-linux-gnu"
	case "arm64":
		archLinuxGnu = "aarch64-linux-gnu"
	case "loongarch64":
		archLinuxGnu = "loongarch64-linux-gnu"
	default:
		archLinuxGnu = "x86_64-linux-gnu"
	}

	execFile := "start.sh"

	d.Build = append(d.Build, "#>>> auto generate by ll-pica begin")

	// 设置 linglong/sources 目录
	d.Build = append(d.Build, []string{
		"# set the linglong/sources directory",
		fmt.Sprintf("SOURCES=\"%s\"", comm.LlSourceDir),
		fmt.Sprintf("export TRIPLET=\"%s\"", archLinuxGnu),
	}...)
	// 如果是应用商店的软件包
	if d.FromAppStore {
		d.PackageKind = "app"
		// linglong/sources 下解压 app 后的目录
		debDirPath := filepath.Join(filepath.Dir(d.Path), "app")

		// // 删除多余的 desktop 文件
		if ret, msg, err := comm.ExecAndWait(10, "sh", "-c",
			fmt.Sprintf("find %s -name '*.desktop' | grep _uos | xargs -I {} rm {}", debDirPath)); err != nil {
			log.Logger.Warnf("remove extra desktop file error: msg: %+v", msg)
		} else {
			log.Logger.Debugf("remove extra desktop file: %+v", ret)
		}

		// 读取desktop 文件
		if ret, msg, err := comm.ExecAndWait(10, "sh", "-c",
			fmt.Sprintf("find %s -name '*.desktop' | grep entries | head -1", debDirPath)); err != nil {
			log.Logger.Errorf("find desktop error: %s out: %s", msg, ret)
		} else {
			log.Logger.Debugf("ret: %+v", ret)
			// 可能出现多个Exec字段，只获取第一个, ret结果有换行符号，替换成空字符串
			execLine, msg, err := comm.ExecAndWait(10, "sh", "-c", fmt.Sprintf("grep Exec= %s | head -1", strings.Replace(ret, "\n", "", -1)))
			if err != nil {
				log.Logger.Errorf("grep \"Exec\" error: %+v, out: %+v", msg, ret)
			} else {
				log.Logger.Debugf("read desktop get Exec %+v", execLine)
			}

			//获取 desktop 文件，Exec 行的内容,并且对字符串做处理
			pattern := regexp.MustCompile(`Exec=|"|\n`)
			execLine = pattern.ReplaceAllLiteralString(execLine, "")
			execSlice := strings.Split(execLine, " ")

			// 切割 Exec 命令
			binPath := strings.Split(execSlice[0], "/")
			// 获取可执行文件的名称
			binFile := binPath[len(binPath)-1]

			// 获取 files 和可执行文件之间路径的字符串
			extractPath := func() string {
				// 查找"files"在路径中的位置
				filesIndex := strings.Index(execSlice[0], "files/")
				if filesIndex == -1 {
					// 如果没有找到"files/"，返回原始路径
					return ""
				}

				// 找到该部分中最后一个斜杠的位置
				part := execSlice[0][filesIndex+len("files/"):]
				lastFolderIndex := strings.LastIndex(part, "/")
				if lastFolderIndex == -1 {
					// 如果没有找到斜杠，返回空
					return ""
				}
				return part[:lastFolderIndex]
			}
			ePath := extractPath()
			execSlice[0] = execFile

			lastIndex := len(execSlice) - 1
			execSlice[lastIndex] = strings.TrimSpace(execSlice[lastIndex])
			newExecLine := strings.Join(execSlice, " ")

			// 提取 Icon 字段
			iconLine, msg, err := comm.ExecAndWait(10, "sh", "-c", fmt.Sprintf("grep \"Icon=\" %s", ret))
			if err != nil {
				log.Logger.Warnf("msg: %+v err:%+v, out: %+v", msg, err, ret)
			} else {
				log.Logger.Debugf("read desktop get Icon %+v", execLine)
			}
			iconSlice := strings.Split(iconLine, "Icon=")
			iconValue := fs.TransIconToLl(iconSlice[1])

			// 玲珑内部的 /opt/apps 路径拼接的是 linglong-id
			d.Build = append(d.Build, []string{
				"export PATH=$PATH:/usr/libexec/linglong/builder/helper",
				"install_dep $SOURCES $PREFIX",
				"# modify desktop, Exec and Icon should not contanin absolut paths",
				"desktopPath=`find $SOURCES/app -name \"*.desktop\" | grep entries`",
				"sed -i '/Exec*/c\\Exec=" + newExecLine + "' $desktopPath",
				"sed -i '/Icon*/c\\Icon=" + iconValue + "' $desktopPath",
				"# use a script as program",
				"echo \"#!/usr/bin/env bash\" > " + execFile,
				"echo \"export LD_LIBRARY_PATH=/opt/apps/" + d.Id + "/files/lib/$TRIPLET:/runtime/lib:/runtime/lib/$TRIPLET:/usr/lib:/usr/lib/$TRIPLET\" >> " + execFile,
				"echo \"cd $PREFIX/" + ePath + " && ./" + binFile + " \\$@\" >> " + execFile,
			}...)

			if ePath == "" {
				d.Command = fmt.Sprintf("/opt/apps/%s/files/%s", d.Id, binFile)
			} else {
				d.Command = fmt.Sprintf("/opt/apps/%s/files/%s/%s", d.Id, ePath, binFile)
			}
		}

		d.Build = append(d.Build, []string{
			"install -d $PREFIX/share",
			"install -d $PREFIX/bin",
			"install -d $PREFIX/lib",
			"install -m 0755 " + execFile + " $PREFIX/bin",
			"# move files",
			"cp -r $SOURCES/app/opt/apps/" + d.Name + "/entries/* $PREFIX/share",
			"cp -r $SOURCES/app/opt/apps/" + d.Name + "/files/* $PREFIX",
		}...)
	} else {
		// TODO
		// 如果不是应用商店的 deb 包
		d.Build = append(d.Build, []string{
			"# move files",
			"cp -r $SOURCES/app/usr/* $PREFIX",
		}...)
	}

	d.Build = append(d.Build, "#>>> auto generate by ll-pica end")
}

// 获取 deb 包
func (d *Deb) GetPackageList() {
	context := cmd.GetContext()
	defer context.Shutdown()
	collectionFactory := context.NewCollectionFactory()
	repo, err := collectionFactory.RemoteRepoCollection().ByName(d.Name)

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

					// 构造下载路径
					task.TempDownPath = filepath.Join(filepath.Dir(d.Path), task.File.Filename)

					// 下载文件
					e = context.Downloader().DownloadWithChecksum(
						context,
						repo.PackageURL(task.File.DownloadURL()).String(),
						task.TempDownPath,
						&task.File.Checksums,
						false)

					source := Source{
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
