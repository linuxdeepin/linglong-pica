/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
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
var SdkConf BaseConfig

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

// initCmd represents the init command
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
					return
				}
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

		for idx, baseInfo := range SdkConf.SdkInfo.Base {
			Logger.Debugf("get %d %s", idx, baseInfo)

			switch baseInfo.Type {
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
					ret, err := CheckFileExits(ConfigInfo.IsoPath)
					if err != nil {
						Logger.Debugf("%v not exists, err: %v", ConfigInfo.IsoPath, err)
					}
					if ret {
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
		ret, err := CheckFileExits(ConfigInfo.Initdir)
		if ret && err == nil && !ConfigInfo.IsInited {
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

		Logger.Debug("end init")
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		Logger.Debug("begin mount iso: ", ConfigInfo.IsoPath)

		ConfigInfo.IsoMountDir = ConfigInfo.Workdir + "/iso/mount"
		err := errors.New("")
		if ret, err := CheckFileExits(ConfigInfo.IsoMountDir); err != nil && !ret {
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

		// mount overlay to base dir
		if ret, err := CheckFileExits(ConfigInfo.RuntimeBasedir + "/files"); err == nil && ret {
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
	Short: "Convert deb to linglong",
	Long: `Convert the deb to linglong For example:
Convert:
	ll-pica init
	ll-pica convert  --config config.yaml --workdir=/mnt/workdir
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
		if ret, err := CheckFileExits(TransInfo.Yamlconfig); err != nil && !ret {
			Logger.Fatal("can not found: ", err)
		} else {
			Logger.Debugf("load: %s", TransInfo.Yamlconfig)
			cacheFd, err := ioutil.ReadFile(TransInfo.Yamlconfig)
			if err != nil {
				Logger.Fatalf("read error: %s %s", err, err)
				return
			}
			err = yaml.Unmarshal(cacheFd, &DebConf)
			if err != nil {
				Logger.Fatalf("unmarshal error: %s", err)
				return
			}
		}

		Logger.Debug("load cache.yaml", TransInfo.CachePath)
		if ret, err := CheckFileExits(TransInfo.CachePath); err == nil && ret {
			// load cache.yaml
			Logger.Debugf("load cache: %s", TransInfo.CachePath)
			cacheFd, err := ioutil.ReadFile(TransInfo.CachePath)
			if err != nil {
				Logger.Warnf("read error: %s %s", err, err)
				return
			}
			err = yaml.Unmarshal(cacheFd, &ConfigInfo)
			if err != nil {
				Logger.Warnf("unmarshal error: %s", err)
				return
			}
		} else {
			Logger.Fatalf("can not found: %s", err)
			return
		}

		ConfigInfo.DebPath = TransInfo.DebPath
		ConfigInfo.Yamlconfig = TransInfo.Yamlconfig
		ConfigInfo.Verbose = TransInfo.Verbose
		ConfigInfo.CachePath = TransInfo.CachePath
		ConfigInfo.DebugMode = TransInfo.DebugMode

		// 创建debdir
		ConfigInfo.DebWorkdir = ConfigInfo.Workdir + "/debdir"
		if ret, err := CheckFileExits(ConfigInfo.DebWorkdir); !ret && err != nil {
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
		if ret, err := CheckFileExits(ConfigInfo.RuntimeBasedir + "/files"); err == nil && ret {
			if ret, err := rfs.MountRfsWithOverlayfs(ConfigInfo.RuntimeBasedir+"/files", ConfigInfo.Workdir+"/iso/live", ConfigInfo.Initdir, ConfigInfo.Basedir, ConfigInfo.Workdir+"/tmpdir", ConfigInfo.Rootfsdir); !ret {
				Logger.Warnf("mount rootfs failed!", err)
			}
		} else {
			if ret, err := rfs.MountRfsWithOverlayfs(ConfigInfo.RuntimeBasedir, ConfigInfo.Workdir+"/iso/live", ConfigInfo.Initdir, ConfigInfo.Basedir, ConfigInfo.Workdir+"/tmpdir", ConfigInfo.Rootfsdir); !ret {
				Logger.Warnf("mount rootfs failed!", err)
			}
		}

		ConfigInfo.MountsItem.DoMountALL()
	},
	Run: func(cmd *cobra.Command, args []string) {
		// fetch deb file
		// DebConfig
		Logger.Debugf("debConfig deb:%v", DebConf.FileElement.Deb)
		for idx, _ := range DebConf.FileElement.Deb {
			// fetch deb file
			if len(DebConf.FileElement.Deb[idx].Ref) > 0 {
				// NOTE: work with go1.15 but feature not sure .
				debFilePath := ConfigInfo.DebWorkdir + "/" + filepath.Base(DebConf.FileElement.Deb[idx].Ref)
				Logger.Warnf("deb file :%s", debFilePath)
				if ret, err := CheckFileExits(debFilePath); err == nil && ret {
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
				}
				Logger.Info("download %s success.", DebConf.FileElement.Deb[idx].Path)
			}
			continue
		}

		// render DebConfig to template save to pica.sh
		// clear pica.sh cache
		picaShellPath := ConfigInfo.DebWorkdir + "/pica.sh"
		if ret, err := CheckFileExits(picaShellPath); err == nil && ret {
			RemovePath(picaShellPath)
		}

		Logger.Infof("render %s script.", picaShellPath)
		RenderDebConfig(DebConf, picaShellPath)

		// exec script in chroot
		Logger.Info("exec script in chroot")
		if ret, msg, err := ChrootExecShell(ConfigInfo.Rootfsdir, picaShellPath, []string{ConfigInfo.DebWorkdir}); !ret {
			Logger.Fatal("exec pica script in chroot failed! :", msg, err)
			return
		} else {
			// 打印详细日志时输出
			LoggerVerbose("exec pica script in chroot output: %s", msg)
		}

		// 定义拷贝的目标目录
		ConfigInfo.ExportDir = ConfigInfo.Workdir + "/" + DebConf.Info.Appid + "/export/runtime"
		// 导出export目录
		ConfigInfo.Export()

		var binReactor BinFormatReactor

		// set files directory
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

		elfLDDLog := ConfigInfo.DebWorkdir + "/elfldd.log"
		elfLDDShell := ConfigInfo.DebWorkdir + "/elfldd.sh"

		// clear history
		if ret, err := CheckFileExits(elfLDDLog); err == nil && ret {
			RemovePath(elfLDDLog)
		}

		if ret, err := CheckFileExits(elfLDDShell); err == nil && ret {
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

		builder := LinglongBuder{
			Appid:       DebConf.Info.Appid,
			Version:     DebConf.Info.Version,
			Description: DebConf.Info.Description,
			Runtime:     "org.deepin.Runtime",
			Rversion:    "",
		}

		// load runtime.json
		Logger.Debugf("loader runtimedir %s", ConfigInfo.RuntimeBasedir)
		builder.LoadRuntimeInfo(ConfigInfo.RuntimeBasedir + "/info.json")

		// run.sh
		// fixme: 依据kind 字段生成 run.sh 的模板

		// FixDesktop()
		ConfigInfo.FixDesktop(DebConf.Info.Appid)

		// update info.json
		CreateInfo(ConfigInfo, &DebConf, builder)

		// 修正版本号
		builder.Version = DebConf.Info.Version

		Logger.Debugf("update linglong builder: %v", builder)

		// create linglong.yaml
		builder.CreateLinglongYamlBuilder(ConfigInfo.ExportDir + "/linglong.yaml")
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

var rootCmd = &cobra.Command{
	Use:   "ll-pica",
	Short: "debian package convert linglong package",
	Long: `Convert the deb to linglong. For example:
Simple:
	ll-pica init -c runtime.yaml -w work-dir
	ll-pica convert -c app.yaml -w work-dir
	ll-pica push -i appid -w work-dir
	ll-pica help
	`,
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
		if ret, err := CheckFileExits(ConfigInfo.AppKeyFile); err != nil && !ret && (ConfigInfo.AppAuthType == AppLoginWithKeyfile) {
			Logger.Errorf("not found keyfile %v, please push with user and password!", ConfigInfo.AppKeyFile)
			ConfigInfo.AppAuthType = AppLoginFailed
			return
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		Logger.Infof("app path %v", ConfigInfo.Workdir+"/"+ConfigInfo.AppId+"/export/runtime")
		appDataPath := ConfigInfo.Workdir + "/" + ConfigInfo.AppId + "/export/runtime"
		if ret, err := CheckFileExits(appDataPath); err != nil && !ret {
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
	},
	PostRun: func(cmd *cobra.Command, args []string) {

	},
}

func main() {
	Logger = InitLog()
	defer Logger.Sync()

	rootCmd.AddCommand(initCmd)
	rootCmd.PersistentFlags().BoolVarP(&ConfigInfo.Verbose, "verbose", "v", false, "verbose output")
	initCmd.Flags().StringVarP(&ConfigInfo.Config, "config", "c", "", "config")
	initCmd.Flags().StringVarP(&ConfigInfo.Workdir, "workdir", "w", "", "work directory")
	err := initCmd.MarkFlagRequired("config")
	if err != nil {
		Logger.Fatal("config required failed", err)
		return
	}

	rootCmd.AddCommand(convertCmd)
	convertCmd.Flags().StringVarP(&TransInfo.Yamlconfig, "config", "c", "", "config")
	convertCmd.Flags().StringVarP(&TransInfo.Workdir, "workdir", "w", "", "work directory")
	err = convertCmd.MarkFlagRequired("config")
	if err != nil {
		Logger.Fatal("yaml config required failed", err)
	}

	if err := convertCmd.MarkFlagRequired("workdir"); err != nil {
		Logger.Fatal("workdir required failed", err)
		return
	}

	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringVarP(&ConfigInfo.AppUsername, "user", "u", "", "username")
	pushCmd.Flags().StringVarP(&ConfigInfo.AppPasswords, "passwords", "p", "", "passwords")
	pushCmd.Flags().StringVarP(&ConfigInfo.AppId, "appid", "i", "", "app id")
	pushCmd.Flags().StringVarP(&ConfigInfo.AppChannel, "channel", "c", "linglong", "app channel")
	pushCmd.Flags().StringVarP(&ConfigInfo.AppRepoUrl, "repo", "r", "", "repo url")
	pushCmd.Flags().StringVarP(&ConfigInfo.AppRepoName, "reponame", "n", "", "repo name")
	pushCmd.Flags().StringVarP(&ConfigInfo.Workdir, "workdir", "w", "", "work directory")
	if err := pushCmd.MarkFlagRequired("workdir"); err != nil {
		Logger.Fatal("workdir required failed", err)
		return
	}

	if err := pushCmd.MarkFlagRequired("appid"); err != nil {
		Logger.Fatal("appid required failed", err)
		return
	}

	rootCmd.CompletionOptions.DisableDefaultCmd = true

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
