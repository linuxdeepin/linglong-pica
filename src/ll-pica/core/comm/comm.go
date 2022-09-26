/*
 * Copyright (c) 2022. Uniontech Software Ltd. All rights reserved.
 *
 * Author: Heysion Y. <heysion@deepin.com>
 *
 * Maintainer: Heysion Y. <heysion@deepin.com>
 *
 * SPDX-License-Identifier: GNU General Public License v3.0 or later
 */
package comm

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	. "ll-pica/utils/fs"
	. "ll-pica/utils/log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// app config with runtime

// var Logger *zap.SugaredLogger

// func init() {
// 	Logger = InitLog()
// }

var ConfigInfo Config
var TransInfo Config
var DebConf DebConfig

type Config struct {
	Verbose           bool `yaml:"verbose"`
	DebugMode         bool
	Config            string `yaml:"config"`
	Workdir           string `yaml:"workdir"`
	Initdir           string `yaml:"initdir"`
	Basedir           string `yaml:"basedir"`
	IsInited          bool   `yaml:"inited"`
	Cache             bool   `yaml:"cache"`
	CachePath         string `yaml:"cache-path"`
	DebWorkdir        string `yaml:"debdir"`
	DebPath           string
	IsRuntimeFetch    bool   `yaml:"runtime-fetched"`
	IsRuntimeCheckout bool   `yaml:"runtime-checkedout"`
	RuntimeOstreeDir  string `yaml:"runtime-ostreedir"`
	RuntimeBasedir    string `yaml:"runtime-basedir"`
	IsIsoDownload     bool   `yaml:"iso-downloaded"`
	IsoPath           string `yaml:"iso-path"`
	IsoMountDir       string `yaml:"iso-mount-dir"`
	IsIsoChecked      bool   `yaml:"iso-checked"`
	Rootfsdir         string `yaml:"rootfsdir"`
	MountsItem        Mounts `yaml:"mounts"`
	Yamlconfig        string
	ExportDir         string `yaml:"exportdir"`
	FilesSearchPath   string `yaml:"files-search-path"`
	BundleKeyFile     string
	BundleAuthType    int8
	BundleUsername    string
	BundlePasswords   string
	BundlePath        string
	BundleRepoUrl     string
	BundleChannel     string
}

func (config *Config) Export() (bool, error) {
	// 检查新建export目录
	if ret, err := CheckFileExits(config.ExportDir); !ret && err != nil {
		CreateDir(config.ExportDir)
	} else {
		os.RemoveAll(config.ExportDir)
		CreateDir(config.ExportDir)
	}

	// 定义需要拷贝的usr目录列表并处理
	usrDirMap := map[string]string{
		"usr/bin":   "files/bin",
		"usr/share": "files/share",
		"usr/lib":   "files/lib",
		"etc":       "files/etc",
	}

	rsyncDir := func(timeout int, src, dst string) (stdout string, stderr string, err error) {
		// 判断rsync命令是否存在
		if _, err := exec.LookPath("rsync"); err != nil {
			// return CopyFileKeepPath(src,dst)
		}
		return ExecAndWait(timeout, "rsync", "-av", src, dst)
	}

	// 处理 initdir
	for key, value := range usrDirMap {
		keyPath := ConfigInfo.Initdir + "/" + key
		valuePath := ConfigInfo.ExportDir + "/" + value
		if ret, err := CheckFileExits(keyPath); ret && err == nil {
			CreateDir(valuePath)
			rsyncDir(30, keyPath+"/", valuePath)
		}
	}

	// 处理 basedir
	for key, value := range usrDirMap {
		keyPath := ConfigInfo.Basedir + "/" + key
		valuePath := ConfigInfo.ExportDir + "/" + value
		if ret, err := CheckFileExits(keyPath); ret && err == nil {
			CreateDir(valuePath)
			rsyncDir(30, keyPath+"/", valuePath)
		}
	}

	// 删除指定文件或者目录
	removeFileList := []string{
		"files/etc/apt/sources.list",
	}
	for _, dir := range removeFileList {
		dirPath := ConfigInfo.ExportDir + "/" + dir
		if ret, err := CheckFileExits(dirPath); ret && err == nil {
			ret, err = RemovePath(dirPath)
			if !ret && err != nil {
				Logger.Errorf("remove %s err! \n", dirPath)
				return false, errors.New("remove path err!")
			}
		}
	}

	// 特殊处理applications、icons、dbus-1、systemd、mime、autostart、help等目录
	specialDirList := []string{
		"files/share/applications", // desktop
		"files/share/icons",
		"files/share/dbus-1",
		"files/lib/systemd",
		"files/share/mime",
		"files/etc/xdg/autostart",
		"files/share/help",
	}
	for _, dir := range specialDirList {
		srcPath := ConfigInfo.ExportDir + "/" + dir + "/"
		if ret, err := CheckFileExits(srcPath); ret && err == nil {
			dstPath := ConfigInfo.ExportDir + "/entries/" + GetFileName(srcPath)
			CreateDir(dstPath)
			rsyncDir(30, srcPath, dstPath)
			os.RemoveAll(srcPath)
		}
	}

	// 拷贝处理/opt目录
	srcOptPath := ConfigInfo.Basedir + "/opt/apps/" + DebConf.Info.Appid
	Logger.Debugf("srcOptPath %s", srcOptPath)
	if ret, err := CheckFileExits(srcOptPath); ret && err == nil {
		rsyncDir(30, srcOptPath+"/", ConfigInfo.ExportDir)
	}

	// 处理icons目录下多余的icon-theme.cache文件
	iconsPath := ConfigInfo.ExportDir + "/entries/icons"
	if ret, err := CheckFileExits(iconsPath); ret && err == nil {
		// 遍历icons下的icon-theme.cache
		filepath.Walk(iconsPath, func(path string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}

			if ret := strings.HasPrefix(f.Name(), "icon-theme.cache"); ret {
				os.RemoveAll(path)
			}

			return nil
		})
	}

	ConfigInfo.FilesSearchPath = ConfigInfo.ExportDir + "/files"
	return true, nil
}

func (config *Config) fixDesktop(desktopFile, appid string) (bool, error) {
	newFileDesktop := GetFilePPath(desktopFile) + "/bak-linglong.desktop"
	newFileDesktop = filepath.Clean(newFileDesktop)

	file, err := os.Open(desktopFile)
	if err != nil {
		Logger.Errorw("desktopFile open failed! : ", desktopFile)
		return false, err
	}
	defer file.Close()

	newFile, newFileErr := os.OpenFile(newFileDesktop, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if newFileErr != nil {
		Logger.Errorw("desktopFile open failed! : ", newFileDesktop)
		return false, newFileErr
	}
	defer newFile.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				Logger.Debug("desktopFile read ok! : ", desktopFile)
				break
			} else {
				Logger.Errorw("read desktopFile failed! : ", desktopFile)
				return false, err
			}
		}
		// 处理Exec
		if strings.HasPrefix(line, "Exec=") {
			valueList := strings.Split(line, "=")
			newLine := strings.TrimRight(valueList[1], "\r\n")
			newLine = TransExecToLl(newLine, appid)
			byteLine := []byte("Exec=" + newLine + "\n")
			newFile.Write(byteLine)
			// 处理TryExec
		} else if strings.HasPrefix(line, "TryExec=") {
			byteLine := []byte("TryExec=" + "\n")
			newFile.Write(byteLine)
			// 处理icon
		} else if strings.HasPrefix(line, "Icon=") {
			valueList := strings.Split(line, "=")
			newLine := strings.TrimRight(valueList[1], "\r\n")
			newLine = TransIconToLl(newLine)
			byteLine := []byte("Icon=" + newLine + "\n")
			newFile.Write(byteLine)
		} else {
			newFile.Write([]byte(line))
		}
	}
	newFile.Sync()

	if ret, err := MoveFileOrDir(newFileDesktop, desktopFile); !ret && err != nil {
		Logger.Errorw("move test.desktop failed!")
		return false, err
	}

	return true, nil
}

func (config *Config) FixDesktop(appid string) (bool, error) {
	applicationsPath := config.ExportDir + "/entries/applications"
	applicationsPath = filepath.Clean(applicationsPath)
	if ret, err := CheckFileExits(applicationsPath); !ret && err != nil {
		Logger.Errorw("applications dir not exists! : ", applicationsPath)
		return false, err
	}

	// 移除desktop目录里面多余文件
	dropfiles := []string{
		"bamf-2.index",
		"mimeinfo.cache",
	}
	for _, file := range dropfiles {
		dropfile := applicationsPath + "/" + file
		if ret, err := CheckFileExits(dropfile); ret && err == nil {
			os.RemoveAll(dropfile)
		}
	}
	// 遍历desktop目录
	fileList, err := ioutil.ReadDir(applicationsPath)
	if err != nil {
		Logger.Errorw("readDir failed! : ", applicationsPath)
		return false, err

	}
	for _, fileinfo := range fileList {
		Logger.Debug("read dir : ", applicationsPath)
		desktopPath := applicationsPath + "/" + fileinfo.Name()
		if ret := strings.HasSuffix(desktopPath, ".desktop"); ret {
			// 处理desktop
			if ok, err := config.fixDesktop(desktopPath, appid); !ok && err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

// 当使用了-w 参数 ，没用 -f 参数时
func (config *Config) FixCachePath() (bool, error) {
	// 检查workdir是否存在
	if ret, err := CheckFileExits(config.Workdir); !ret || err != nil {
		return false, err
	}
	retWork := strings.HasPrefix(config.Workdir, "/mnt/workdir")
	retCache := strings.HasPrefix(config.CachePath, "/mnt/workdir")
	if !retWork && retCache {
		config.CachePath = config.Workdir + "/cache.yaml"
	}
	return true, nil
}

type MountItem struct {
	MountPoint string `yaml:"mountpoint"`
	Source     string `yaml:"source"`
	Type       string `yaml:"type"`
	TypeLive   string `yaml:"typelive"`
	IsRbind    bool   `yaml:"bind"`
}

type Mounts struct {
	Mounts map[string]MountItem `yaml:"mounts"`
}

func (ts Mounts) DoMountALL() []error {

	Logger.Debug("mount list: ", len(ts.Mounts))
	var errs []error
	if len(ts.Mounts) == 0 {
		return errs
	}

	var msg string
	var err error

	for _, item := range ts.Mounts {

		Logger.Debugf("mount: ", item.MountPoint, item.Source, item.Type, item.TypeLive, item.IsRbind)
		if IsRbind := item.IsRbind; IsRbind {

			// sudo mount --rbind /tmp/ /mnt/workdir/rootfs/tmp/
			_, msg, err = ExecAndWait(10, "mount", "--rbind", item.Source, item.MountPoint)
			if err != nil {
				Logger.Warnf("mount bind failed: ", msg, err)
				errs = append(errs, err)
				// continue
			}

		} else if item.TypeLive != "" {
			_, msg, err = ExecAndWait(10, "mount", item.TypeLive, "-t", item.Type, item.MountPoint)
			if err != nil {
				Logger.Warnf("mount failed: ", msg, err)
				errs = append(errs, err)
			}
		} else {
			_, msg, err = ExecAndWait(10, "mount", "-t", item.Type, item.Source, item.MountPoint)
			if err != nil {
				Logger.Warnf("mount failed: ", msg, err)
				errs = append(errs, err)
			}
		}

	}
	return errs
}

func (ts Mounts) DoUmountALL() []error {
	Logger.Debug("mount list: ", len(ts.Mounts))
	var errs []error
	if len(ts.Mounts) == 0 {
		return errs
	}

	for _, item := range ts.Mounts {
		Logger.Debugf("umount: ", item.MountPoint)
		_, msg, err := ExecAndWait(10, "umount", item.MountPoint)
		if err != nil {
			Logger.Warnf("umount failed: ", msg, err)
			errs = append(errs, err)
		} else {
			delete(ts.Mounts, item.MountPoint)
		}

	}
	return errs
}

func (ts Mounts) DoUmountAOnce() []error {
	return nil
	Logger.Debug("mount list: ", len(ts.Mounts))
	var errs []error
	if len(ts.Mounts) == 0 {
		return nil
	}

	idx := 0
UMOUNT_ONCE:
	_, msg, err := ExecAndWait(10, "umount", "-R", ConfigInfo.Rootfsdir)
	if err == nil {
		idx++
		if idx < 10 {
			goto UMOUNT_ONCE
		}
	} else {
		Logger.Warnf("umount success: ", msg, err)
		errs = append(errs, nil)
	}
	for _, item := range ts.Mounts {
		Logger.Debugf("umount: ", item.MountPoint)
		delete(ts.Mounts, item.MountPoint)

	}
	return errs
}

func (ts *Mounts) FillMountRules() {

	Logger.Debug("mount list: ", len(ts.Mounts))
	ts.Mounts[ConfigInfo.Rootfsdir+"/dev/pts"] = MountItem{ConfigInfo.Rootfsdir + "/dev/pts", "", "devpts", "devpts-live", false}
	ts.Mounts[ConfigInfo.Rootfsdir+"/sys"] = MountItem{ConfigInfo.Rootfsdir + "/sys", "", "sysfs", "sysfs-live", false}
	ts.Mounts[ConfigInfo.Rootfsdir+"/proc"] = MountItem{ConfigInfo.Rootfsdir + "/proc", "", "proc", "proc-live", false}
	ts.Mounts[ConfigInfo.Rootfsdir+"/tmp/"] = MountItem{ConfigInfo.Rootfsdir + "/tmp", "/tmp", "", "", true}
	ts.Mounts[ConfigInfo.Rootfsdir+"/etc/resolv.conf"] = MountItem{ConfigInfo.Rootfsdir + "/etc/resolv.conf", "/etc/resolv.conf", "", "", true}

	Logger.Debug("mount list: ", len(ts.Mounts))
}

// exec and wait for command
func ExecAndWait(timeout int, name string, arg ...string) (stdout, stderr string, err error) {
	Logger.Debugf("cmd: %s %+v\n", name, arg)
	cmd := exec.Command(name, arg...)
	var bufStdout, bufStderr bytes.Buffer
	cmd.Stdout = &bufStdout
	cmd.Stderr = &bufStderr
	err = cmd.Start()
	if err != nil {
		err = fmt.Errorf("start fail: %w", err)
		return
	}

	// wait for process finished
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		if err = cmd.Process.Kill(); err != nil {
			err = fmt.Errorf("timeout: %w", err)
			return
		}
		<-done
		err = fmt.Errorf("time out and process was killed")
	case err = <-done:
		stdout = bufStdout.String()
		stderr = bufStderr.String()
		if err != nil {
			err = fmt.Errorf("run: %w", err)
			return
		}
	}
	return
}

// deb config info struct
type DebConfig struct {
	Info struct {
		Appid       string `yaml:"appid"`
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
		Kind        string `yaml:"kind"`
		Arch        string `yaml:"arch"`
	} `yaml:"info"`
	FileElement struct {
		Deb     []DebInfo `yaml:"deb"`
		Package []string  `yaml:"add-package"`
	} `yaml:"file"`
	ChrootInfo struct {
		PreCmd  string `yaml:"pre-command"`
		PostCmd string `yaml:"post-command"`
	} `yaml:"chroot"`
}

func (ts *DebConfig) MergeInfo(t *DebConfig) bool {
	if ts.Info.Appid == "" {
		ts.Info.Appid = t.Info.Appid
	}
	if ts.Info.Name == "" {
		ts.Info.Name = t.Info.Name
	}
	if ts.Info.Version == "" {
		ts.Info.Version = t.Info.Version
	}
	if ts.Info.Description == "" {
		ts.Info.Description = t.Info.Description
	}
	if ts.Info.Kind == "" {
		ts.Info.Kind = t.Info.Kind
	}
	if ts.Info.Arch == "" {
		ts.Info.Arch = t.Info.Arch
	}
	return true
}

type DebInfo struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Ref  string `yaml:"ref"`
	Hash string `yaml:"hash"`
	Path string
}

func (ts *DebInfo) CheckDebHash() bool {
	if ts.Hash == "" {
		return false
	}
	hash, err := GetFileSha256(ts.Path)
	if err != nil {
		Logger.Warn(err)
		return false
	}
	if hash == ts.Hash {
		return true
	}

	return false
}

// FetchDebFile
func (ts *DebInfo) FetchDebFile(dstPath string) bool {

	Logger.Debugf("FetchDebFile %s,ts:%v", dstPath, ts)

	if ts.Type == "repo" {

		CreateDir(GetFilePPath(dstPath))

		if ret, msg, err := ExecAndWait(1<<20, "wget", "-O", dstPath, ts.Ref); err != nil {

			Logger.Warnf("msg: %+v err:%+v, out: %+v", msg, err, ret)
			return false
		} else {
			Logger.Debugf("ret: %+v", ret)
		}

		if ret, err := CheckFileExits(dstPath); ret {
			ts.Path = dstPath
			return true
		} else {
			Logger.Warnf("downalod %s , err:%+v", dstPath, err)
			return false
		}
	}
	return false
}

type BaseConfig struct {
	SdkInfo struct {
		Base  []BaseInfo `yaml:"base"`
		Extra ExtraInfo  `yaml:"extra"`
	} `yaml:"sdk"`
}

type BaseInfo struct {
	Type   string `yaml:"type"`
	Ref    string `yaml:"ref"`
	Hash   string `yaml:"hash"`
	Remote string `yaml:"remote"`
	Path   string
}

func (ts *BaseInfo) CheckIsoHash() bool {
	if ts.Hash == "" {
		return false
	}
	hash, err := GetFileSha256(ts.Path)
	if err != nil {
		Logger.Warn(err)
		return false
	}
	if hash == ts.Hash {
		return true
	}

	return false
}

func (ts *BaseInfo) FetchIsoFile(workdir, isopath string) bool {
	//转化绝对路径
	isoAbsPath, _ := filepath.Abs(isopath)
	//如果下载目录不存在就创建目录
	CreateDir(GetFilePPath(isoAbsPath))
	if ts.Type == "iso" {
		ts.Path = isoAbsPath

		if ret, msg, err := ExecAndWait(1<<20, "wget", "-O", ts.Path, ts.Ref); err != nil {
			Logger.Warnf("msg: %+v err:%+v, out: %+v", msg, err, ret)
			return false
		} else {
			Logger.Debugf("ret: %+v", ret)
		}

		if ret, err := CheckFileExits(ts.Path); ret {
			return true
		} else {
			Logger.Warnf("downalod %s , err:%+v", ts.Path, err)
			return false
		}
	}
	return false
}

func (ts *BaseInfo) CheckoutOstree(target string) bool {
	// ConfigInfo.RuntimeBasedir = fmt.Sprintf("%s/runtimedir", ConfigInfo.Workdir)
	Logger.Debug("ostree checkout %s to %s", ts.Path, target)
	_, msg, err := ExecAndWait(10, "ostree", "checkout", "--repo", ts.Path, ts.Ref, target)

	if err != nil {
		Logger.Errorf("msg: %v ,err: %+v", msg, err)
		return false
	}
	return true
}

func (ts *BaseInfo) InitOstree(ostreePath string) bool {
	if ts.Type == "ostree" {
		Logger.Debug("ostree init")
		ts.Path = ostreePath
		_, msg, err := ExecAndWait(10, "ostree", "init", "--mode=bare-user-only", "--repo", ts.Path)
		if err != nil {
			Logger.Errorf("msg: %v ,err: %+v", msg, err)
			return false
		}
		Logger.Debug("ostree remote add", ts.Remote)

		_, msg, err = ExecAndWait(10, "ostree", "remote", "add", "runtime", ts.Remote, "--repo", ts.Path, "--no-gpg-verify")
		if err != nil {
			Logger.Errorf("msg: %+v err:%+v", msg, err)
			return false
		}

		Logger.Debug("ostree pull")
		_, msg, err = ExecAndWait(300, "ostree", "pull", "runtime", "--repo", ts.Path, "--mirror", ts.Ref)
		if err != nil {
			Logger.Errorf("msg: %+v err:%+v", msg, err)
			return false
		}

		return true
	}
	return false
}

type ExtraInfo struct {
	Repo    []string `yaml:"repo"`
	Package []string `yaml:"package"`
	Cmd     string   `yaml:"command"`
}

func (ts *ExtraInfo) WriteRootfsRepo(config Config) bool {
	if ret, err := CheckFileExits(config.Rootfsdir + "/etc/apt/sources.list"); !ret && err != nil {
		Logger.Warnf("rootfs sources.list not exists ! ,err : %+v", err)
		return false
	}
	file, err := os.OpenFile(config.Rootfsdir+"/etc/apt/sources.list", os.O_RDWR|os.O_APPEND|os.O_TRUNC, 0644)
	if err != nil {
		Logger.Warnf("open sources.list failed! err: %+v", err)
		return false
	}
	defer file.Close()
	for _, value := range ts.Repo {
		if _, err := file.WriteString(value + "\n"); err != nil {
			Logger.Warnf("write sources.list failed! err : %+v", err)
			return false
		}
	}
	file.Sync()

	return true
}

type ExtraShellTemplate struct {
	ExtraCommand string
	Verbose      bool
}

const EXTRA_COMMAND_TMPL = `#!/bin/bash
{{if .Verbose }}set -x {{end}}
function extra_command {
    {{if len .ExtraCommand }}{{.ExtraCommand}}{{end}}
    echo extra_command
}
{{if len .ExtraCommand }}extra_command{{end}}
echo init
`

func (ts *ExtraInfo) RenderExtraShell(save string) (bool, error) {
	tpl, err := template.New("init").Parse(EXTRA_COMMAND_TMPL)

	if err != nil {
		Logger.Fatalf("parse deb shell template failed! ", err)
		return false, nil
	}

	extraShell := ExtraShellTemplate{"", ConfigInfo.Verbose}

	// PostCommand
	Logger.Debugf("cmd: %s", ts.Cmd)
	if len(ts.Cmd) > 0 {
		extraShell.ExtraCommand = ts.Cmd
	}

	// create save file
	Logger.Debug("extra shell save file: ", save)
	saveFd, ret := os.Create(save)
	if ret != nil {
		Logger.Warnf("save to %s failed!", save)
		return false, ret
	}
	defer saveFd.Close()

	// render template
	Logger.Debug("render template: ", extraShell)
	tpl.Execute(saveFd, extraShell)

	return true, nil
}

func GetFileSha256(filename string) (string, error) {
	Logger.Debug("GetFileSha256 :", filename)
	hasher := sha256.New()
	s, err := ioutil.ReadFile(filename)
	if err != nil {
		Logger.Warn(err)
		return "", err
	}
	_, err = hasher.Write(s)
	if err != nil {
		Logger.Warn(err)
		return "", err
	}

	sha256Sum := hex.EncodeToString(hasher.Sum(nil))
	Logger.Debug("file hash: ", sha256Sum)

	return sha256Sum, nil
}

func UmountPath(path string) bool {
	Logger.Debugf("umount path: %s", path)
	if ret, msg, err := ExecAndWait(10, "umount", path); err != nil {
		Logger.Debugf("umount path failed: %s %v \nout:%s", msg, err, ret)
		return false
	} else {
		Logger.Debugf("umount path %s \nout:%s", msg, ret)
		return true
	}
}

const (
	BundleLoginFailed       int8 = -1
	BundleLoginWithPassword int8 = iota
	BundleLoginWithKeyfile
)

// Bundle push with ll-builder
func LinglongBuilderWarp(t int8, conf *Config) (bool, error) {
	// ll-builder push --repo-url  http://repo-dev.linglong.space --channel linglong *.uab
	// max wait time for two MTL
	BundleCommand := []string{
		"push",
		"--repo-url", conf.BundleRepoUrl,
		"--channel", conf.BundleChannel,
	}
	Logger.Debugf("command args: %v", BundleCommand)
	switch t {
	case BundleLoginWithPassword:
		BundleCommand = append(BundleCommand, []string{
			"--username",
			conf.BundleUsername,
			"--password",
			conf.BundlePasswords}...)
		break
	case BundleLoginWithKeyfile:
		BundleCommand = append(BundleCommand, []string{
			"--auth",
			conf.BundleKeyFile}...)
		break
	default:
		return false, fmt.Errorf("not support")
	}

	BundleCommand = append(BundleCommand, conf.BundlePath)

	// ll-builder push
	// ll-builder wait max timeout 3600 seconds wtf
	if ret, msg, err := ExecAndWait(1<<12, "ll-builder", BundleCommand...); err == nil {
		// if ConfigInfo.Verbose {
		// 	Logger.Infof("output: %v", ret)
		// }
		LoggerVerbose("output: %v", ret)
		return true, nil
	} else {
		Logger.Debugf("output: %v", ret, msg)
		return false, err
	}

}

// logger verbose
func LoggerVerbose(k string, s ...interface{}) {
	if ConfigInfo.Verbose || ConfigInfo.DebugMode {
		Logger.Infof(k, s)
	}
}
