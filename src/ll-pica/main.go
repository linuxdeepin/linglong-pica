/*
 * Copyright (c) 2022. Uniontech Software Ltd. All rights reserved.
 *
 * Author: Heysion Y. <heysion@deepin.com>
 *
 * Maintainer: Heysion Y. <heysion@deepin.com>
 *
 * SPDX-License-Identifier: GNU General Public License v3.0 or later
 */
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "ll-pica/core"
	. "ll-pica/core/comm"
	. "ll-pica/core/info"
	. "ll-pica/core/linglong"
	. "ll-pica/utils/fs"
	. "ll-pica/utils/log"
	"ll-pica/utils/rfs"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var disableDevelop string

// var Logger *zap.SugaredLogger

// var RootfsMountList Mounts

var SdkConf BaseConfig

// initCmd represents the init command

func SetOverlayfs(lower string, upper string, workdir string) error {
	Logger.Debug("SetOverlayfs :", lower, upper, workdir)
	// mount lower dir to upper dir
	//mount -t overlay overlay -o lowerdir=$WORK_DIR/lower,upperdir=$WORK_DIR/upper,workdir=$WORK_DIR/work $WORK_DIR/merged
	tempDir := ConfigInfo.Workdir + "/temp"
	err := os.Mkdir(tempDir, 0755)
	if os.IsNotExist(err) {
		Logger.Error("mkdir failed: ", err)
		return err
	}
	msg := fmt.Sprintf("lowerdir=%s:%s,upperdir=%s,workdir=%s", lower, upper, workdir, tempDir)
	_, msg, err = ExecAndWait(10, "mount", "-t", "overlay", "overlay", "-o", msg, ConfigInfo.Rootfsdir)
	if err != nil {
		Logger.Error("mount overlayfs failed: ", msg, err)
	}
	return nil
}

func UmountOverlayfs(workdir string) error {
	Logger.Debug("UmountOverlayfs :", workdir)
	// umount upper dir
	_, msg, err := ExecAndWait(10, "umount", workdir)
	if err != nil {
		Logger.Error("umount overlayfs failed: ", msg, err)
	}
	return nil
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "init sdk runtime env",
	Long:  `init sdk runtime env with iso and runtime .`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// 转换获取路径为绝对路径
		if configPath, err := filepath.Abs(ConfigInfo.Config); err != nil {
			Logger.Errorf("Trans %s err: %s ", ConfigInfo.Config, err)
		} else {
			ConfigInfo.Config = configPath
		}

		if workPath, err := filepath.Abs(ConfigInfo.Workdir); err != nil {
			Logger.Errorf("Trans %s err: %s ", ConfigInfo.Workdir, err)
		} else {
			ConfigInfo.Workdir = workPath
		}

		Logger.Debug("begin process cache: ", ConfigInfo.Cache)
		configCache := ConfigInfo.Workdir + "/cache.yaml"
		runtimeDir := ConfigInfo.Workdir + "/runtime"
		isoDir := ConfigInfo.Workdir + "/iso"

		ClearRuntime := func() {
			Logger.Debug("begin clear runtime")
			if _, err := os.Stat(runtimeDir); !os.IsNotExist(err) {
				Logger.Debugf("remove runtime: %s", runtimeDir)
				err = os.RemoveAll(runtimeDir)
				if err != nil {
					Logger.Errorf("remove error", err)
				}

			}
		}

		ClearIso := func() {
			Logger.Debug("begin clear iso")
			if ret, _ := CheckFileExits(isoDir); ret {
				Logger.Debugf("remove iso: %s", isoDir)
				// err = os.RemoveAll(isoDir)
				// if err != nil {
				// 	Logger.Errorf("remove error", err)
				// }
			}
		}

		if _, err := os.Stat(configCache); !os.IsNotExist(err) && ConfigInfo.Cache {
			// load cache.yaml
			Logger.Debugf("load: %s", configCache)
			cacheFd, err := ioutil.ReadFile(configCache)
			if err != nil {
				Logger.Warnf("read error: %s", err)
				return
			}
			err = yaml.Unmarshal(cacheFd, &ConfigInfo)
			if err != nil {
				Logger.Warnf("unmarshal error: %s", err)
				return
			}
			Logger.Debugf("load cache.yaml success: %s", configCache)

			Logger.Debug("clear runtime: ", ConfigInfo.IsRuntimeFetch)
			if !ConfigInfo.IsRuntimeFetch {
				// fixme:(heysion) double fetch with ostree cached
				ClearRuntime()
			}

			err = os.Mkdir(runtimeDir, 0755)
			if err != nil {
				Logger.Info("create runtime dir error: ", err)
			}

			Logger.Debug("clear iso: ", ConfigInfo.IsIsoDownload)
			if !ConfigInfo.IsIsoDownload {
				// fixme:(heysion) dobule fetch iso with this
				ClearIso()
			}

			err = os.Mkdir(isoDir, 0755)
			if err != nil {
				Logger.Warn("create iso dir error: ", err)
			}

			return // Config Cache exist
		} else {
			Logger.Debug("Config Cache not exist")
			if !ConfigInfo.IsRuntimeCheckout {
				err := os.RemoveAll(ConfigInfo.RuntimeBasedir)
				if err != nil {
					Logger.Errorf("remove error", err)
				}
			}

			return // Config Cache not exist
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		Logger.Debug("begin run: ", ConfigInfo.Config)
		if ConfigInfo.Config != "" {
			yamlFile, err := ioutil.ReadFile(ConfigInfo.Config)
			if err != nil {
				Logger.Errorf("get %s error: %v", ConfigInfo.Config, err)
				return
			}
			err = yaml.Unmarshal(yamlFile, &SdkConf)
			if err != nil {
				Logger.Errorf("error: %v", err)
				return
			}
		}

		for idx, context := range SdkConf.SdkInfo.Base {
			Logger.Debugf("get %d %s", idx, context)
			switch context.Type {
			case "ostree":
				if !ConfigInfo.IsRuntimeFetch {
					Logger.Debugf("ostree init %s", ConfigInfo.IsRuntimeFetch)

					ConfigInfo.RuntimeOstreeDir = ConfigInfo.Workdir + "/runtime"
					if ret := SdkConf.SdkInfo.Base[idx].InitOstree(ConfigInfo.RuntimeOstreeDir); !ret {
						Logger.Warn("init ostree failed")
						ConfigInfo.IsRuntimeFetch = false
						continue
					} else {
						ConfigInfo.IsRuntimeFetch = true
					}

					ConfigInfo.RuntimeBasedir = ConfigInfo.Workdir + "/runtimedir"
					if ret := SdkConf.SdkInfo.Base[idx].CheckoutOstree(ConfigInfo.RuntimeBasedir); !ret {
						Logger.Warn("checkout ostree failed")
						ConfigInfo.IsRuntimeCheckout = false
						continue
					} else {
						ConfigInfo.IsRuntimeCheckout = true
					}

				}
				continue

			case "iso":

				if !ConfigInfo.IsIsoDownload {
					Logger.Debugf("iso download %s", ConfigInfo.IsIsoDownload)

					ConfigInfo.IsoPath = ConfigInfo.Workdir + "/iso/base.iso"

					if ret, _ := CheckFileExits(ConfigInfo.IsoPath); ret {
						SdkConf.SdkInfo.Base[idx].Path = ConfigInfo.IsoPath
						if ret := SdkConf.SdkInfo.Base[idx].CheckIsoHash(); !ret {
							ConfigInfo.IsIsoChecked = false
							RemovePath(ConfigInfo.IsoPath)
							SdkConf.SdkInfo.Base[idx].Path = ""
						} else {
							Logger.Debugf("download skipped because of %s cached", ConfigInfo.IsoPath)
							ConfigInfo.IsIsoChecked = true
							continue
						}
					}

					if ret := SdkConf.SdkInfo.Base[idx].FetchIsoFile(ConfigInfo.Workdir, ConfigInfo.IsoPath); !ret {
						ConfigInfo.IsIsoDownload = false
						Logger.Errorf("download iso failed")
						return
					} else {
						ConfigInfo.IsIsoDownload = true
					}
					Logger.Debug("iso download success")
				}

				if !ConfigInfo.IsIsoChecked {
					Logger.Debug("iso check hash")
					if ret := SdkConf.SdkInfo.Base[idx].CheckIsoHash(); !ret {
						ConfigInfo.IsIsoChecked = false
						Logger.Errorf("check iso hash failed")
						return
					} else {
						ConfigInfo.IsIsoChecked = true
					}
					Logger.Debug("iso check hash success")
				}
				continue
			}
		}

		ConfigInfo.Initdir = ConfigInfo.Workdir + "/initdir"
		Logger.Debug("set initdir: ", ConfigInfo.Initdir)
		// 不读取缓存文件时，需清理initdir
		if ret, err := CheckFileExits(ConfigInfo.Initdir); ret && err == nil && !ConfigInfo.IsInited {
			ret, err = RemovePath(ConfigInfo.Initdir)
			if !ret || err != nil {
				Logger.Errorf("failed to remove %s\n", ConfigInfo.Initdir)
			}
			ret, err = CreateDir(ConfigInfo.Initdir)
			if !ret || err != nil {
				Logger.Errorf("failed to create %s\n", ConfigInfo.Initdir)
			}
		} else {
			ret, err = CreateDir(ConfigInfo.Initdir)
			if !ret || err != nil {
				Logger.Errorf("failed to create %s\n", ConfigInfo.Initdir)
			}
		}

		// extra
		Logger.Debug("extra init")
		// if SdkConf.SdkInfo.Extra != ExtraInfo{,} {
		// 	Logger.Debug(SdkConf.SdkInfo.Extra)
		// }

		Logger.Debug("end init")
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		Logger.Debugf("config :%+v", ConfigInfo)
		err := errors.New("")
		Logger.Debug("begin mount iso: ", ConfigInfo.IsoPath)
		ConfigInfo.IsoMountDir = ConfigInfo.Workdir + "/iso/mount"

		if ret, _ := CheckFileExits(ConfigInfo.IsoMountDir); !ret {
			err = os.Mkdir(ConfigInfo.IsoMountDir, 0755)
			if os.IsNotExist(err) {
				Logger.Error("mkdir iso mount dir failed!", err)
			}
		}

		var msg string
		_, msg, err = ExecAndWait(10, "mount", "-o", "loop", ConfigInfo.IsoPath, ConfigInfo.IsoMountDir)
		if err != nil {
			Logger.Warnf("mount iso failed!", msg, err)
		}
		UmountIsoDir := func() {
			ExecAndWait(10, "umount", ConfigInfo.IsoMountDir)
		}

		defer UmountIsoDir()

		// mount squashfs to base dir

		baseDir := ConfigInfo.Workdir + "/iso/live"
		err = os.Mkdir(baseDir, 0755)
		if os.IsNotExist(err) {
			Logger.Error("mkdir iso mount dir failed!", err)
		}
		_, msg, err = ExecAndWait(10, "mount", ConfigInfo.IsoMountDir+"/live/filesystem.squashfs", baseDir)
		if err != nil {
			Logger.Error("mount squashfs failed!", msg, err)
		}
		UmountSquashfsDir := func() {
			ExecAndWait(10, "umount", baseDir)
		}
		defer UmountSquashfsDir()

		ConfigInfo.Rootfsdir = ConfigInfo.Workdir + "/rootfs"
		err = os.Mkdir(ConfigInfo.Rootfsdir, 0755)
		if os.IsNotExist(err) {
			Logger.Error("mkdir rootfsdir failed!", err)
		}

		if os.IsNotExist(err) {
			Logger.Error("mkdir runtime basedir failed!", err)
		}

		// mount overlay to base dir
		if ret, _ := CheckFileExits(ConfigInfo.RuntimeBasedir + "/files"); ret {
			SetOverlayfs(baseDir, ConfigInfo.RuntimeBasedir+"/files", ConfigInfo.Initdir)
		} else {
			SetOverlayfs(baseDir, ConfigInfo.RuntimeBasedir, ConfigInfo.Initdir)
		}

		UmountRootfsDir := func() {
			ExecAndWait(10, "umount", ConfigInfo.Rootfsdir)
		}
		defer UmountRootfsDir()

		ConfigInfo.MountsItem.FillMountRules()

		fmt.Printf("Inside rootCmd PostRun with args: %v\n", args)
		ConfigInfo.IsInited = true

		yamlData, err := yaml.Marshal(&ConfigInfo)
		if err != nil {
			Logger.Errorf("convert to yaml failed!")
		}
		// Logger.Debugf("write Config Cache: %v", string(yamlData))
		err = ioutil.WriteFile(fmt.Sprintf("%s/cache.yaml", ConfigInfo.Workdir), yamlData, 0644)
		if err != nil {
			Logger.Error("write cache.yaml failed!")
		}

		ConfigInfo.MountsItem.DoMountALL()
		defer ConfigInfo.MountsItem.DoUmountALL()

		// write source.list
		Logger.Debugf("Start write sources.list !")
		if ret := SdkConf.SdkInfo.Extra.WriteRootfsRepo(ConfigInfo); !ret {
			Logger.Errorf("Write sources.list failed!")
		}

		// write extra shell
		if len(SdkConf.SdkInfo.Extra.Cmd) > 0 {

			SdkConf.SdkInfo.Extra.RenderExtraShell(ConfigInfo.Rootfsdir + "/init.sh")

			defer func() { RemovePath(ConfigInfo.Rootfsdir + "/init.sh") }()

			ChrootExecShellBare(ConfigInfo.Rootfsdir, ConfigInfo.Rootfsdir+"/init.sh")
		}
	},
}

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert deb to uab",
	Long: `Convert the deb to uab For example:
Convert:
	ll-pica init
	ll-pica convert --deb abc.deb --config config.yaml --workdir=/mnt/workdir
	`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		Logger.Debugf("workdir %s ,cache file %s", TransInfo.Workdir, TransInfo.CachePath)

		if TransInfo.CachePath == "" {
			if ret, err := CheckFileExits(TransInfo.Workdir + "/cache.yaml"); !ret {
				Logger.Fatal("cache-file required failed", err)
				return
			}
			TransInfo.CachePath = TransInfo.Workdir + "/cache.yaml"
		}
		// fmt.Printf("Inside rootCmd PersistentPreRun with args: %v\n", args)
	},
	PreRun: func(cmd *cobra.Command, args []string) {

		if ConfigInfo.Verbose {
			Logger.Info("verbose mode enabled")
			TransInfo.Verbose = true
		}

		// 转换获取路径为绝对路径
		if yamlPath, err := filepath.Abs(TransInfo.Yamlconfig); err != nil {
			Logger.Errorf("Trans %s err: %s ", TransInfo.Yamlconfig, err)
		} else {
			TransInfo.Yamlconfig = yamlPath
		}

		if workPath, err := filepath.Abs(TransInfo.Workdir); err != nil {
			Logger.Errorf("Trans %s err: %s ", TransInfo.Workdir, err)
		} else {
			TransInfo.Workdir = workPath
		}

		if cachePath, err := filepath.Abs(TransInfo.CachePath); err != nil {
			Logger.Errorf("Trans %s err: %s ", TransInfo.CachePath, err)
		} else {
			TransInfo.CachePath = cachePath
		}

		// 修复CachePath参数
		if ret, err := TransInfo.FixCachePath(); !ret || err != nil {
			Logger.Fatal("can not found: ", TransInfo.Workdir)
		}

		Logger.Debug("load yaml config", TransInfo.Yamlconfig)

		if ret, msg := CheckFileExits(TransInfo.Yamlconfig); !ret {
			Logger.Fatal("can not found: ", msg)
		} else {
			Logger.Debugf("load: %s", TransInfo.Yamlconfig)
			cacheFd, err := ioutil.ReadFile(TransInfo.Yamlconfig)
			if err != nil {
				Logger.Fatalf("read error: %s %s", err, msg)
				return
			}
			// Logger.Debugf("load: %s", cacheFd)
			err = yaml.Unmarshal(cacheFd, &DebConf)
			if err != nil {
				Logger.Fatalf("unmarshal error: %s", err)
				return
			}
			// Logger.Debugf("loaded %+v", DebConf)

		}

		Logger.Debug("load cache.yaml", TransInfo.CachePath)

		if ret, msg := CheckFileExits(TransInfo.CachePath); ret {
			// load cache.yaml
			Logger.Debugf("load cache: %s", TransInfo.CachePath)
			cacheFd, err := ioutil.ReadFile(TransInfo.CachePath)
			if err != nil {
				Logger.Warnf("read error: %s %s", err, msg)
				return
			}

			err = yaml.Unmarshal(cacheFd, &ConfigInfo)
			if err != nil {
				Logger.Warnf("unmarshal error: %s", err)
				return
			}
			// Logger.Debugf("load config info %v", ConfigInfo)
		} else {
			Logger.Fatalf("can not found: %s", msg)
			return
		}

		// fmt.Printf("Inside rootCmd PreRun with args: %v\n", args)
		//Logger.Debug("mount all", ConfigInfo.MountsItem)

		Logger.Debug("configinfo.rootfsdir", ConfigInfo.Rootfsdir)

		ConfigInfo.DebPath = TransInfo.DebPath
		ConfigInfo.Yamlconfig = TransInfo.Yamlconfig
		ConfigInfo.Verbose = TransInfo.Verbose
		ConfigInfo.CachePath = TransInfo.CachePath
		ConfigInfo.DebugMode = TransInfo.DebugMode

		// 创建debdir
		ConfigInfo.DebWorkdir = ConfigInfo.Workdir + "/debdir"
		if ret, err := CheckFileExits(ConfigInfo.DebWorkdir); ret && err == nil {
			// ret, err = RemovePath(ConfigInfo.DebWorkdir)
			// if !ret || err != nil {
			// 	Logger.Errorf("failed to remove %s\n", ConfigInfo.DebWorkdir)
			// }
			// ret, err = CreateDir(ConfigInfo.DebWorkdir)
			// if !ret || err != nil {
			// 	Logger.Errorf("failed to create %s\n", ConfigInfo.DebWorkdir)
			// }
		} else {
			ret, err = CreateDir(ConfigInfo.DebWorkdir)
			if !ret || err != nil {
				Logger.Errorf("failed to create %s\n", ConfigInfo.DebWorkdir)
			}
		}

		// 新建basedir
		ConfigInfo.Basedir = ConfigInfo.Workdir + "/basedir"
		if ret, err := CheckFileExits(ConfigInfo.Basedir); ret && err == nil {
			ret, err = RemovePath(ConfigInfo.Basedir)
			if !ret || err != nil {
				Logger.Errorf("failed to remove %s\n", ConfigInfo.Basedir)
			}
			ret, err = CreateDir(ConfigInfo.Basedir)
			if !ret || err != nil {
				Logger.Errorf("failed to create %s\n", ConfigInfo.Basedir)
			}
		} else {
			ret, err = CreateDir(ConfigInfo.Basedir)
			if !ret || err != nil {
				Logger.Errorf("failed to create %s\n", ConfigInfo.Basedir)
			}
		}

		if ret, err := rfs.MountIso(ConfigInfo.Workdir+"/iso/mount", ConfigInfo.IsoPath); !ret {
			Logger.Warnf("mount iso failed!", err)
		}

		if ret, err := rfs.MountSquashfs(ConfigInfo.Workdir+"/iso/live", ConfigInfo.Workdir+"/iso/mount/live/filesystem.squashfs"); !ret {
			Logger.Warnf("mount live failed!", err)
		}

		// mount overlay to base dir
		Logger.Debug("Rootfsdir:", ConfigInfo.Rootfsdir, "runtimeBasedir:", ConfigInfo.RuntimeBasedir, "basedir:", ConfigInfo.Basedir, "workdir:", ConfigInfo.Workdir)

		CreateDir(ConfigInfo.Workdir + "/tmpdir")
		if ret, _ := CheckFileExits(ConfigInfo.RuntimeBasedir + "/files"); ret {
			if ret, err := rfs.MountRfsWithOverlayfs(ConfigInfo.RuntimeBasedir+"/files", ConfigInfo.Workdir+"/iso/live", ConfigInfo.Initdir, ConfigInfo.Basedir, ConfigInfo.Workdir+"/tmpdir", ConfigInfo.Rootfsdir); !ret {
				Logger.Warnf("mount rootfs failed!", err)
			}
		} else {
			if ret, err := rfs.MountRfsWithOverlayfs(ConfigInfo.RuntimeBasedir, ConfigInfo.Workdir+"/iso/live", ConfigInfo.Initdir, ConfigInfo.Basedir, ConfigInfo.Workdir+"/tmpdir", ConfigInfo.Rootfsdir); !ret {
				Logger.Warnf("mount rootfs failed!", err)
			}
		}

		ConfigInfo.MountsItem.DoMountALL()

		//  umount ConfigInfo.Rootfsdir

	},
	Run: func(cmd *cobra.Command, args []string) {
		// fmt.Printf("Inside rootCmd Run with args: %v\n", args)

		// check enter deb file
		Logger.Debug("check DebPath:", ConfigInfo.DebPath)
		if ret, msg := CheckFileExits(ConfigInfo.DebPath); !ret {
			Logger.Warnf("can not found: ", msg)
		}

		// fetch deb file
		// DebConfig
		Logger.Debugf("debConfig deb:%v", DebConf.FileElement.Deb)
		for idx, _ := range DebConf.FileElement.Deb {
			// fetch deb file
			if len(DebConf.FileElement.Deb[idx].Ref) > 0 {
				// NOTE: work with go1.15 but feature not sure .
				debFilePath := ConfigInfo.DebWorkdir + "/" + filepath.Base(DebConf.FileElement.Deb[idx].Ref)
				Logger.Warnf("deb file :%s", debFilePath)
				if ret, _ := CheckFileExits(debFilePath); ret {
					DebConf.FileElement.Deb[idx].Path = debFilePath
					if ret := DebConf.FileElement.Deb[idx].CheckDebHash(); ret {
						Logger.Infof("download skipped because of %s cached", debFilePath)
						continue
					} else {
						RemovePath(debFilePath)
						DebConf.FileElement.Deb[idx].Path = ""
					}
				}
				// fetch deb file
				DebConf.FileElement.Deb[idx].FetchDebFile(debFilePath)
				Logger.Debugf("fetch deb path:[%d] %s", idx, debFilePath)
				// check deb hash
				if ret := DebConf.FileElement.Deb[idx].CheckDebHash(); !ret {
					Logger.Warnf("check deb hash failed! : ", DebConf.FileElement.Deb[idx].Name)
					continue
				} else {
					Logger.Info("download %s success.", DebConf.FileElement.Deb[idx].Path)
				}

			} else {
				continue
			}
		}

		// render DebConfig to template save to pica.sh
		// Logger.Debugf("render berfore %+v:", DebConf)
		// clear pica.sh cache
		picaShellPath := ConfigInfo.DebWorkdir + "/pica.sh"
		if ret, _ := CheckFileExits(picaShellPath); ret {
			RemovePath(picaShellPath)
		}

		Logger.Infof("render %s script.", picaShellPath)
		RenderDebConfig(DebConf, picaShellPath)

		// chroot
		Logger.Info("exec script in chroot")
		if ret, msg, err := ChrootExecShell(ConfigInfo.Rootfsdir, picaShellPath, []string{ConfigInfo.DebWorkdir}); !ret {
			Logger.Fatal("exec pica script in chroot failed! :", msg, err)
			return
		} else {
			LoggerVerbose("exec pica script in chroot output: %s", msg)
		}

		// copy deb data
		// fixme(heysion): todo

		// fixme(jianqiang)
		// make new directory that need to be created for linglong files stucturesk
		// 定义拷贝的目标目录
		ConfigInfo.ExportDir = ConfigInfo.Workdir + "/" + DebConf.Info.Appid + "/export/runtime"
		// 导出export目录
		ConfigInfo.Export()

		// find all elf file with path
		// FilerList := ("libc.so","lib.so")

		var binReactor BinFormatReactor

		// fixme(heysion) set files directory
		binReactor.SearchPath = ConfigInfo.FilesSearchPath
		// get elf binary  need exclude with self path

		binReactor.GetElfList(binReactor.SearchPath + "/lib")

		excludeSoList := []string{
			"ld-linux-aarch64.so",
			"ld-linux-armhf.so",
			"ld-linux-x86-64.so",
			"ld-linux.so",
			"ld64.so",
			"libc.so",
			"libc.so.6",
			"libm.so.6",
			"libdl.so",
			"libdl.so.2",
			"libgcc_s.so",
			"libgcc_s.so.1",
			"libm.so",
			"libstdc++.so",
			"libstdc++.so.6",
			"libz.so.1",
			"libXi.so.6",
			"libX11.so.6",
			"libX11-xcb.so.1",
			"libselinux.so.1",
			"libnssutil3.so",
			"libnss3.so",
		}
		Logger.Debugf("exclude so list:", excludeSoList)

		// check  dlopen if it exists append depends to list
		Logger.Debug("call GetEntryDlopenList:")
		binReactor.GetEntryDlopenList(excludeSoList)
		Logger.Debug("call GetEntryDlopenList: %v", binReactor.ElfEntrySoPath)

		//binReactor.FixElfLDDPath(binReactor.SearchPath + "bin/lib")
		//
		// GetFindElfMissDepends(ConfigInfo.Basedir + "/lib")
		elfLDDLog := ConfigInfo.DebWorkdir + "/elfldd.log"
		elfLDDShell := ConfigInfo.DebWorkdir + "/elfldd.sh"

		// clear history
		if ret, _ := CheckFileExits(elfLDDLog); ret {
			RemovePath(elfLDDLog)
		}

		if ret, _ := CheckFileExits(elfLDDShell); ret {
			RemovePath(elfLDDShell)
		}

		Logger.Debugf("out: %s , sh: %s", elfLDDLog, elfLDDShell)

		binReactor.RenderElfWithLDD(elfLDDLog, elfLDDShell)

		// chroot
		if ret, msg, err := ChrootExecShell(ConfigInfo.Rootfsdir, elfLDDShell, []string{ConfigInfo.FilesSearchPath}); !ret {
			Logger.Fatal("chroot exec shell failed:", msg, err)
			return
		}

		// check result with chroot exec shell
		if ret, err := CheckFileExits(elfLDDLog); !ret {
			Logger.Fatal("chroot exec shell failed:", ret, err)
			return
		}

		// read elfldd.log
		Logger.Debug("read elfldd.log", elfLDDLog)
		if elfLDDLogFile, err := os.Open(elfLDDLog); err != nil {
			Logger.Fatal("open elfldd.log failed:", err)
			return
		} else {
			defer elfLDDLogFile.Close()

			binReactor.ElfNeedPath = make(map[string]uint)

			LogFileItor := bufio.NewScanner(elfLDDLogFile)
			LogFileItor.Split(bufio.ScanLines)
			var ReadLine string
			//var ReadLines map[string]uint = make(map[string]uint)
			for LogFileItor.Scan() {
				ReadLine = LogFileItor.Text()
				if len(ReadLine) > 0 {
					binReactor.ElfNeedPath[ReadLine] = 1
				}
			}

			binReactor.FixElfNeedPath(excludeSoList)

			Logger.Debugf("fix exclude so list: %v", binReactor.ElfNeedPath)

		}

		Logger.Debugf("found %d elf need objects", len(binReactor.ElfNeedPath))

		binReactor.CopyElfNeedPath(ConfigInfo.Rootfsdir, ConfigInfo.FilesSearchPath)

		// fix library
		// if msg, ret := GetElfNeedWithLDD("/bin/bash"); ret != nil {
		// 	Logger.Debug("get elf need failed: ", msg)
		// }

		builder := LinglongBuder{}

		builder.Appid = DebConf.Info.Appid
		builder.Version = DebConf.Info.Version
		builder.Description = DebConf.Info.Description
		builder.Runtime = "org.deepin.Runtime"
		builder.Rversion = "20.6"

		// load runtime.json
		Logger.Debugf("loader runtimedir %s", ConfigInfo.RuntimeBasedir)
		builder.LoadRuntimeInfo(ConfigInfo.RuntimeBasedir + "/info.json")

		// run.sh
		// fixme:(heysion) 依据kind 字段生成 run.sh 的模板

		// fix desktop
		// FixDesktop()
		ConfigInfo.FixDesktop(DebConf.Info.Appid)

		// update info.json
		CreateInfo(ConfigInfo, &DebConf, builder)

		// 修正版本号
		builder.Version = DebConf.Info.Version

		// umount

		Logger.Debugf("update linglong builder: %v", builder)

		// create linglong.yaml
		builder.CreateLinglongYamlBuilder(ConfigInfo.ExportDir + "/linglong.yaml")

		// build uab
		// ll-builder export --local
		//builder.CreateLinglongBuilder(ConfigInfo.ExportDir)
		//builder.LinglongExport(ConfigInfo.ExportDir)

	},
	PostRun: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Inside rootCmd PostRun with args: %v\n", args)
		ConfigInfo.MountsItem.DoUmountALL()

		// umount overlayfs
		Logger.Debug("umount rootfs")
		if ret, err := rfs.UmountRfs(ConfigInfo.Rootfsdir); !ret {
			Logger.Warnf("umount rootfs failed!", err)
		}

		// umount squashfs
		Logger.Debug("umount squashfs")
		if ret, err := rfs.UmountSquashfs(ConfigInfo.Workdir + "/iso/live"); !ret {
			Logger.Warnf("umount squashfs failed!", err)
		}

		// umount iso
		Logger.Debug("umount iso")
		if ret, err := rfs.UmountIso(ConfigInfo.Workdir + "/iso/mount"); !ret {
			Logger.Warnf("umount iso failed!", err)
		}

	},
}

// rootCmd represents the convert command
var rootCmd = &cobra.Command{
	Use:   "ll-pica",
	Short: "debian package convert linglong package",
	Long: `Convert the deb to uab. For example:
Simple:
	ll-pica init 
	ll-pica convert -d abc.deb --config config.yaml -w /mnt/workdir
	ll-pica help


	`,
	// Uncomment the following line if your bare application
	// has an action associated with it:

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(cmd.Use, "1.0.1")
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "push app to repo",
	Long: `Push app to repo that used ll-builder push For example:
push:
	ll-pica push -u deepin -p deepin -i appid -w workdir
	`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	PreRun: func(cmd *cobra.Command, args []string) {
		Logger.Infof("parse input app:", ConfigInfo.AppId)

		// 转化工作目录为绝对路径
		if workPath, err := filepath.Abs(ConfigInfo.Workdir); err != nil {
			Logger.Errorf("Trans %s err: %s ", ConfigInfo.Workdir, err)
		} else {
			ConfigInfo.Workdir = workPath
		}

		// auth username
		if ConfigInfo.AppUsername == "" || ConfigInfo.AppPasswords == "" {
			ConfigInfo.AppAuthType = AppLoginWithKeyfile
		} else {
			Logger.Infof("app login with password")
			ConfigInfo.AppAuthType = AppLoginWithPassword
		}

		// AppKeyFile path
		ConfigInfo.AppKeyFile = GetHomePath() + "/.linglong/.user.json"
		// keyfile
		if ret, _ := CheckFileExits(ConfigInfo.AppKeyFile); !ret && ConfigInfo.AppAuthType == AppLoginWithKeyfile {
			Logger.Errorf("not found keyfile %v, please push with user and password!", ConfigInfo.AppKeyFile)
			ConfigInfo.AppAuthType = AppLoginFailed
			return

		} else {
			ConfigInfo.AppAuthType = AppLoginWithKeyfile
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		Logger.Infof("app path %v", ConfigInfo.Workdir+"/"+ConfigInfo.AppId+"/export/runtime")
		appDataPath := ConfigInfo.Workdir + "/" + ConfigInfo.AppId + "/export/runtime"
		if ret, _ := CheckFileExits(appDataPath); !ret {
			Logger.Errorf("app data dir not exist : %v", appDataPath)
			return
		}
		// 执行上传操作
		// 获取当前路径
		cwdPath, err := os.Getwd()
		if err != nil {
			Logger.Errorf("get cwd path Failed %v", err)
			return
		}
		// 进入appDataPath
		err = os.Chdir(appDataPath)
		if err != nil {
			Logger.Errorf("chdir failed: %s", err)
			return
		}

		if ret, err := LinglongBuilderWarp(ConfigInfo.AppAuthType, &ConfigInfo); !ret {
			Logger.Errorf("%v push failed: %v", appDataPath, err)
			return
		}
		// 退出appDatapath
		err = os.Chdir(cwdPath)
		if err != nil {
			Logger.Errorf("chdir failed: %s", err)
			return
		}

		// if ConfigInfo.BundlePath == "" {
		// 	if workdirPath, err := os.Getwd(); err != nil {
		// 		Logger.Debugf("get working directory: %v", err)
		// 		return
		// 	} else {
		// 		Logger.Debugf("working directory: %v", workdirPath)
		// 		if bundleList, err := FindBundlePath(workdirPath); err == nil {
		// 			Logger.Debugf("found bundle file %v", bundleList)
		// 			// mutiple bundles

		// 			for _, bundle := range bundleList {
		// 				ConfigInfo.BundlePath = bundle

		// 				if ret, err := LinglongBuilderWarp(ConfigInfo.BundleAuthType, &ConfigInfo); !ret {
		// 					Logger.Infof("push failed: %v", err, bundle)
		// 					continue
		// 				}
		// 			}

		// 			return

		// 		} else {
		// 			Logger.Errorf("not found bundle")
		// 			return
		// 		}
		// 	}
		// } else {
		// 	//
		// 	if ret, _ := CheckFileExits(ConfigInfo.BundlePath); ret {
		// 		if ret := HasBundleName(ConfigInfo.BundlePath); ret {
		// 			// run push

		// 			if ret, err := LinglongBuilderWarp(ConfigInfo.BundleAuthType, &ConfigInfo); !ret {
		// 				Logger.Errorf("push failed: %v", err)
		// 				return
		// 			}

		// 		} else {
		// 			Logger.Errorf("need bundle file %s", ConfigInfo.BundlePath)
		// 			return
		// 		}
		// 	} else {
		// 		Logger.Errorf("not found bundle %s", ConfigInfo.BundlePath)
		// 		return
		// 	}
		// }
	},
	PostRun: func(cmd *cobra.Command, args []string) {

	},
}

func main() {

	Logger = InitLog()
	defer Logger.Sync()

	// init cmd add

	rootCmd.AddCommand(initCmd)
	rootCmd.PersistentFlags().BoolVarP(&ConfigInfo.Verbose, "verbose", "v", false, "verbose output")

	initCmd.Flags().StringVarP(&ConfigInfo.Config, "config", "c", "", "config")
	initCmd.Flags().StringVarP(&ConfigInfo.Workdir, "workdir", "w", "", "work directory")
	initCmd.Flags().BoolVarP(&ConfigInfo.Cache, "keep-cached", "k", true, "keep cached")
	//initCmd.Flags().BoolVarP(&ConfigInfo.Verbose, "verbose", "v", false, "verbose output")

	err := initCmd.MarkFlagRequired("config")
	if err != nil {
		Logger.Fatal("config required failed", err)
		return
	}

	// convert cmd add
	rootCmd.AddCommand(convertCmd)
	convertCmd.Flags().StringVarP(&TransInfo.Yamlconfig, "config", "c", "", "config")
	convertCmd.Flags().StringVarP(&TransInfo.Workdir, "workdir", "w", "", "work directory")
	// convertCmd.Flags().StringVarP(&TransInfo.CachePath, "cache-file", "f", "", "cache yaml file")
	convertCmd.Flags().StringVarP(&TransInfo.DebPath, "deb-file", "d", "", "deb file")
	//convertCmd.Flags().BoolVarP(&TransInfo.Verbose, "verbose", "v", false, "verbose output")

	err = convertCmd.MarkFlagRequired("config")
	if err != nil {
		Logger.Fatal("yaml config required failed", err)
	}

	if err := convertCmd.MarkFlagRequired("workdir"); err != nil {
		Logger.Fatal("workdir required failed", err)
		return
	}

	// err = convertCmd.MarkFlagRequired("deb-file")
	// if err != nil {
	// 	Logger.Fatal("deb file required failed", err)
	// }

	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringVarP(&ConfigInfo.AppUsername, "username", "u", "", "username")
	pushCmd.Flags().StringVarP(&ConfigInfo.AppPasswords, "passwords", "p", "", "passwords")
	pushCmd.Flags().StringVarP(&ConfigInfo.AppId, "appid", "i", "", "app id")
	pushCmd.Flags().StringVarP(&ConfigInfo.AppChannel, "channel", "c", "linglong", "app channel")
	pushCmd.Flags().StringVarP(&ConfigInfo.AppRepoUrl, "repo", "r", "", "repo url")
	pushCmd.Flags().StringVarP(&ConfigInfo.Workdir, "workdir", "w", "", "work directory")

	if err := pushCmd.MarkFlagRequired("workdir"); err != nil {
		Logger.Fatal("workdir required failed", err)
		return
	}

	if err := pushCmd.MarkFlagRequired("appid"); err != nil {
		Logger.Fatal("appid required failed", err)
		return
	}
	//pushCmd.Flags().BoolVarP(&ConfigInfo.Verbose, "verbose", "v", false, "verbose output")

	//pushCmd.MarkFlagsMutuallyExclusive("keyfile", "username")

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// root cmd add
	//pushCmd.PersistentFlags().BoolVarP(&ConfigInfo.Verbose, "verbose", "v", false, "verbose output")

	// if ConfigInfo.Workdir == "" {
	// 	ConfigInfo.Workdir = "/mnt/workdir"
	// }

	// if TransInfo.Workdir == "" {
	// 	TransInfo.Workdir = "/mnt/workdir"
	// }
	// // fix cache path
	// if TransInfo.CachePath == "" {
	// 	TransInfo.CachePath = TransInfo.Workdir + "/cache.yaml"
	// }

	// go build -ldflags '-X ll-pica/utils/log.disableLogDebug=yes -X main.disableDevelop=yes'
	// fmt.Printf("disableDevelop: %s\n", disableDevelop)
	if disableDevelop != "" {
		Logger.Debugf("develop mode disable")
		TransInfo.DebugMode = false
		ConfigInfo.DebugMode = false
	} else {
		Logger.Debugf("develop mode enabled")
		TransInfo.DebugMode = true
		ConfigInfo.DebugMode = true
		// debug mode enable verbose mode
		TransInfo.Verbose = true
		ConfigInfo.Verbose = true
	}

	ConfigInfo.MountsItem.Mounts = make(map[string]MountItem)

	err = rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
