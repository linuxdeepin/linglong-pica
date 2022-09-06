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
	. "ll-pica/utils/fs"
	. "ll-pica/utils/log"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// type BaseSdk struct {
// 	Sdk map[string]dataset `yaml:"sdk"`
// }
// type dataset []DataSet

// type DataSet struct {
// 	Type   string `yaml:"type"`
// 	Ref    string `yaml:"ref"`
// 	Hash   string `yaml:"hash"`
// 	Remote string `yaml:"remote"`
// }

var logger *zap.SugaredLogger

// var RootfsMountList Mounts

var SdkConf BaseConfig

// initCmd represents the init command

func SetOverlayfs(lower string, upper string, workdir string) error {
	logger.Debug("SetOverlayfs :", lower, upper, workdir)
	// mount lower dir to upper dir
	//mount -t overlay overlay -o lowerdir=$WORK_DIR/lower,upperdir=$WORK_DIR/upper,workdir=$WORK_DIR/work $WORK_DIR/merged
	tempDir := ConfigInfo.Workdir + "/temp"
	err := os.Mkdir(tempDir, 0755)
	if os.IsNotExist(err) {
		logger.Error("mkdir failed: ", err)
		return err
	}
	msg := fmt.Sprintf("lowerdir=%s:%s,upperdir=%s,workdir=%s", upper, lower, workdir, tempDir)
	_, msg, err = ExecAndWait(10, "mount", "-t", "overlay", "overlay", "-o", msg, ConfigInfo.Rootfsdir)
	if err != nil {
		logger.Error("mount overlayfs failed: ", msg, err)
	}
	return nil
}

func UmountOverlayfs(workdir string) error {
	logger.Debug("UmountOverlayfs :", workdir)
	// umount upper dir
	_, msg, err := ExecAndWait(10, "umount", workdir)
	if err != nil {
		logger.Error("umount overlayfs failed: ", msg, err)
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
			logger.Errorf("Trans %s err: %s ", ConfigInfo.Config, err)
		} else {
			ConfigInfo.Config = configPath
		}

		if workPath, err := filepath.Abs(ConfigInfo.Workdir); err != nil {
			logger.Errorf("Trans %s err: %s ", ConfigInfo.Workdir, err)
		} else {
			ConfigInfo.Workdir = workPath
		}

		logger.Debug("begin process cache: ", ConfigInfo.Cache)
		configCache := ConfigInfo.Workdir + "/cache.yaml"
		runtimeDir := ConfigInfo.Workdir + "/runtime"
		isoDir := ConfigInfo.Workdir + "/iso"

		ClearRuntime := func() {
			logger.Debug("begin clear runtime")
			if _, err := os.Stat(runtimeDir); !os.IsNotExist(err) {
				logger.Debugf("remove runtime: %s", runtimeDir)
				err = os.RemoveAll(runtimeDir)
				if err != nil {
					logger.Errorf("remove error", err)
				}

			}
		}

		ClearIso := func() {
			logger.Debug("begin clear iso")
			if _, err := os.Stat(isoDir); !os.IsNotExist(err) {
				logger.Debugf("remove iso: %s", isoDir)
				err = os.RemoveAll(isoDir)
				if err != nil {
					logger.Errorf("remove error", err)
				}
			}
		}

		if _, err := os.Stat(configCache); !os.IsNotExist(err) && ConfigInfo.Cache {
			// load cache.yaml
			logger.Debugf("load: %s", configCache)
			cacheFd, err := ioutil.ReadFile(configCache)
			if err != nil {
				logger.Warnf("read error: %s", err)
				return
			}
			err = yaml.Unmarshal(cacheFd, &ConfigInfo)
			if err != nil {
				logger.Warnf("unmarshal error: %s", err)
				return
			}
			logger.Debugf("load cache.yaml success: %s", configCache)

			logger.Debug("clear runtime: ", ConfigInfo.IsRuntimeFetch)
			if !ConfigInfo.IsRuntimeFetch {
				ClearRuntime()
			}

			err = os.Mkdir(runtimeDir, 0755)
			if err != nil {
				logger.Info("create runtime dir error: ", err)
			}

			logger.Debug("clear iso: ", ConfigInfo.IsIsoDownload)
			if !ConfigInfo.IsIsoDownload {
				ClearIso()
			}

			err = os.Mkdir(isoDir, 0755)
			if err != nil {
				logger.Info("create iso dir error: ", err)
			}

			return // Config Cache exist
		} else {
			logger.Debug("Config Cache not exist")
			if !ConfigInfo.IsRuntimeCheckout {
				err := os.RemoveAll(ConfigInfo.RuntimeBasedir)
				if err != nil {
					logger.Errorf("remove error", err)
				}
			}

			return // Config Cache not exist
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debug("begin run: ", ConfigInfo.Config)
		if ConfigInfo.Config != "" {
			yamlFile, err := ioutil.ReadFile(ConfigInfo.Config)
			if err != nil {
				logger.Errorf("get %s error: %v", ConfigInfo.Config, err)
				return
			}
			err = yaml.Unmarshal(yamlFile, &SdkConf)
			if err != nil {
				logger.Errorf("error: %v", err)
				return
			}
		}

		for idx, context := range SdkConf.SdkInfo.Base {
			logger.Debugf("get %d %s", idx, context)
			switch context.Type {
			case "ostree":
				if !ConfigInfo.IsRuntimeFetch {
					logger.Debug("ostree init")
					ConfigInfo.RuntimeOstreeDir = ConfigInfo.Workdir + "/runtime"
					if ret := SdkConf.SdkInfo.Base[idx].InitOstree(ConfigInfo.RuntimeOstreeDir); !ret {
						logger.Error("init ostree failed")
						ConfigInfo.IsRuntimeFetch = false
						return
					} else {
						ConfigInfo.IsRuntimeFetch = true
					}

					ConfigInfo.RuntimeBasedir = ConfigInfo.Workdir + "/runtimedir"
					if ret := SdkConf.SdkInfo.Base[idx].CheckoutOstree(ConfigInfo.RuntimeBasedir); !ret {
						logger.Error("checkout ostree failed")
						ConfigInfo.IsRuntimeCheckout = false
						return
					} else {
						ConfigInfo.IsRuntimeCheckout = true
					}

				}
				continue

			case "iso":

				if !ConfigInfo.IsIsoDownload {
					logger.Debug("iso download")

					ConfigInfo.IsoPath = ConfigInfo.Workdir + "/iso/base.iso"

					if ret := SdkConf.SdkInfo.Base[idx].FetchIsoFile(ConfigInfo.Workdir, ConfigInfo.IsoPath); !ret {
						ConfigInfo.IsIsoDownload = false
						logger.Errorf("download iso failed")
						return
					} else {
						ConfigInfo.IsIsoDownload = true
					}
					logger.Debug("iso download success")
				}

				if !ConfigInfo.IsIsoChecked {
					logger.Debug("iso check hash")
					if ret := SdkConf.SdkInfo.Base[idx].CheckIsoHash(); !ret {
						ConfigInfo.IsIsoChecked = false
						logger.Errorf("check iso hash failed")
						return
					} else {
						ConfigInfo.IsIsoChecked = true
					}
					logger.Debug("iso check hash success")
				}
				continue
			}
		}

		ConfigInfo.Basedir = ConfigInfo.Workdir + "/basedir"
		logger.Debug("set basedir: ", ConfigInfo.Basedir)

		// extra
		logger.Debug("extra init")
		// if SdkConf.SdkInfo.Extra != ExtraInfo{,} {
		// 	logger.Debug(SdkConf.SdkInfo.Extra)
		// }

		logger.Debug("end init")
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		logger.Debugf("config :%+v", ConfigInfo)
		err := errors.New("")
		logger.Debug("begin mount iso: ", ConfigInfo.IsoPath)
		ConfigInfo.IsoMountDir = ConfigInfo.Workdir + "/iso/mount"

		if ret, _ := CheckFileExits(ConfigInfo.IsoMountDir); !ret {
			err = os.Mkdir(ConfigInfo.IsoMountDir, 0755)
			if os.IsNotExist(err) {
				logger.Error("mkdir iso mount dir failed!", err)
			}
		}

		var msg string
		_, msg, err = ExecAndWait(10, "mount", "-o", "loop", ConfigInfo.IsoPath, ConfigInfo.IsoMountDir)
		if err != nil {
			logger.Error("mount iso failed!", msg, err)
		}
		// UmountIsoDir := func() {
		// 	ExecAndWait(10, "umount", ConfigInfo.IsoMountDir)
		// }

		// defer UmountIsoDir()

		// mount squashfs to base dir

		baseDir := ConfigInfo.Workdir + "/iso/live"
		err = os.Mkdir(baseDir, 0755)
		if os.IsNotExist(err) {
			logger.Error("mkdir iso mount dir failed!", err)
		}
		_, msg, err = ExecAndWait(10, "mount", ConfigInfo.IsoMountDir+"/live/filesystem.squashfs", baseDir)
		if err != nil {
			logger.Error("mount squashfs failed!", msg, err)
		}
		// UmountSquashfsDir := func() {
		// 	ExecAndWait(10, "umount", baseDir)
		// }
		// defer UmountSquashfsDir()

		ConfigInfo.Rootfsdir = ConfigInfo.Workdir + "/rootfs"
		err = os.Mkdir(ConfigInfo.Rootfsdir, 0755)
		if os.IsNotExist(err) {
			logger.Error("mkdir rootfsdir failed!", err)
		}

		err = os.Mkdir(ConfigInfo.Basedir, 0755)

		if os.IsNotExist(err) {
			logger.Error("mkdir runtime basedir failed!", err)
		}

		// mount overlay to base dir
		SetOverlayfs(baseDir, ConfigInfo.RuntimeBasedir, ConfigInfo.Basedir)

		ConfigInfo.MountsItem.FillMountRules()

		fmt.Printf("Inside rootCmd PostRun with args: %v\n", args)
		ConfigInfo.IsInited = true

		yamlData, err := yaml.Marshal(&ConfigInfo)
		if err != nil {
			logger.Errorf("convert to yaml failed!")
		}
		// logger.Debugf("write Config Cache: %v", string(yamlData))
		err = ioutil.WriteFile(fmt.Sprintf("%s/cache.yaml", ConfigInfo.Workdir), yamlData, 0644)
		if err != nil {
			logger.Error("write cache.yaml failed!")
		}

		ConfigInfo.MountsItem.DoMountALL()

		// write source.list
		logger.Debugf("Start write sources.list !")
		if ret := SdkConf.SdkInfo.Extra.WriteRootfsRepo(ConfigInfo); !ret {
			logger.Errorf("Write sources.list failed!")
		}
		ConfigInfo.MountsItem.DoUmountAOnce()
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
		// fmt.Printf("Inside rootCmd PersistentPreRun with args: %v\n", args)
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		// 转换获取路径为绝对路径
		if yamlPath, err := filepath.Abs(TransInfo.Yamlconfig); err != nil {
			logger.Errorf("Trans %s err: %s ", TransInfo.Yamlconfig, err)
		} else {
			TransInfo.Yamlconfig = yamlPath
		}

		if workPath, err := filepath.Abs(TransInfo.Workdir); err != nil {
			logger.Errorf("Trans %s err: %s ", TransInfo.Workdir, err)
		} else {
			TransInfo.Workdir = workPath
		}

		if cachePath, err := filepath.Abs(TransInfo.CachePath); err != nil {
			logger.Errorf("Trans %s err: %s ", TransInfo.CachePath, err)
		} else {
			TransInfo.CachePath = cachePath
		}

		if TransInfo.Verbose {
			fmt.Println(TransInfo.Verbose)
		}

		logger.Debug("load yaml config", TransInfo.Yamlconfig)

		if ret, msg := CheckFileExits(TransInfo.Yamlconfig); !ret {
			logger.Fatal("can not found: ", msg)
		} else {
			logger.Debugf("load: %s", TransInfo.Yamlconfig)
			cacheFd, err := ioutil.ReadFile(TransInfo.Yamlconfig)
			if err != nil {
				logger.Fatalf("read error: %s %s", err, msg)
				return
			}
			// logger.Debugf("load: %s", cacheFd)
			err = yaml.Unmarshal(cacheFd, &DebConf)
			if err != nil {
				logger.Fatalf("unmarshal error: %s", err)
				return
			}
			// logger.Debugf("loaded %+v", DebConf)

		}

		logger.Debug("load cache.yaml", TransInfo.CachePath)

		if ret, msg := CheckFileExits(TransInfo.CachePath); ret {
			// load cache.yaml
			logger.Debugf("load cache: %s", TransInfo.CachePath)
			cacheFd, err := ioutil.ReadFile(TransInfo.CachePath)
			if err != nil {
				logger.Warnf("read error: %s %s", err, msg)
				return
			}

			err = yaml.Unmarshal(cacheFd, &ConfigInfo)
			if err != nil {
				logger.Warnf("unmarshal error: %s", err)
				return
			}
			// logger.Debugf("load config info %v", ConfigInfo)
		} else {
			logger.Fatalf("can not found: %s", msg)
			return
		}

		// fmt.Printf("Inside rootCmd PreRun with args: %v\n", args)
		//logger.Debug("mount all", ConfigInfo.MountsItem)

		// todo(ll-pica init umount?)
		//ConfigInfo.MountsItem.DoMountALL()

		logger.Debug("configinfo.rootfsdir", ConfigInfo.Rootfsdir)

		ConfigInfo.DebPath = TransInfo.DebPath
		ConfigInfo.Yamlconfig = TransInfo.Yamlconfig
		ConfigInfo.Verbose = TransInfo.Verbose
		ConfigInfo.CachePath = TransInfo.CachePath
		// ConfigInfo.Workdir = TransInfo.Workdir
		// ConfigInfo.Rootfsdir = TransInfo.Rootfsdir

		// logger.Debug("ConfigInfo:", ConfigInfo)
		// logger.Debug("TransInfo:", TransInfo)
		// logger.Debug("DebConf:", DebConf)

		// if TransInfo.DebWorkdir != "" && ConfigInfo.DebWorkdir == "" {
		// 	ConfigInfo.DebWorkdir = TransInfo.DebWorkdir
		// }

		if ConfigInfo.DebWorkdir == "" {
			ConfigInfo.DebWorkdir = ConfigInfo.Workdir + "/debdir"
			if ret, _ := CheckFileExits(ConfigInfo.DebWorkdir); !ret {
				CreateDir(ConfigInfo.DebWorkdir)
			}
		}

		// todo(ll-pica init umount?)
		// if ret, err := rfs.MountIso(ConfigInfo.IsoPath, ConfigInfo.Workdir+"/iso/mount"); !ret {
		// 	logger.Error("mount iso failed!", err)
		// }

		// if ret, err := rfs.MountSquashfs(ConfigInfo.Workdir+"/iso/live", ConfigInfo.Workdir+"/iso/mount/live/filesystem.squashfs"); !ret {
		// 	logger.Error("mount iso failed!", err)
		// }

		// mount overlay to base dir
		logger.Debug("Rootfsdir:", ConfigInfo.Rootfsdir, "runtimeBasedir:", ConfigInfo.RuntimeBasedir, "basedir:", ConfigInfo.Basedir, "workdir:", ConfigInfo.Workdir)

		// todo(ll-pica init umount?)
		// CreateDir(ConfigInfo.Workdir + "/tmpdir")
		// if ret, err := rfs.MountRfsWithOverlayfs(ConfigInfo.Basedir, ConfigInfo.Rootfsdir, ConfigInfo.RuntimeBasedir, ConfigInfo.Workdir+"/tmpdir", ConfigInfo.Workdir+"/iso/live"); !ret {
		// 	logger.Error("mount iso failed!", err)
		// }

		//  umount ConfigInfo.Rootfsdir

	},
	Run: func(cmd *cobra.Command, args []string) {
		// fmt.Printf("Inside rootCmd Run with args: %v\n", args)

		// check enter deb file
		logger.Debug("check DebPath:", ConfigInfo.DebPath)
		if ret, msg := CheckFileExits(ConfigInfo.DebPath); !ret {
			logger.Debug("can not found: ", msg)
		}

		// fetch deb file
		// DebConfig
		logger.Debugf("debConfig deb:%v", DebConf.FileElement.Deb)
		for idx, _ := range DebConf.FileElement.Deb {
			// fetch deb file
			DebConf.FileElement.Deb[idx].FetchDebFile(ConfigInfo.DebWorkdir)
			logger.Debugf("fetch deb path:[%d] %s", idx, DebConf.FileElement.Deb[idx].Path)
			// check deb hash
			if ret := DebConf.FileElement.Deb[idx].CheckDebHash(); !ret {
				logger.Fatal("check deb hash failed! : ", DebConf.FileElement.Deb[idx].Name)
				return
			}
		}

		// render DebConfig to template save to pica.sh
		logger.Debugf("render berfore %s:", DebConf.FileElement.Deb)
		RenderDebConfig(DebConf, ConfigInfo.DebWorkdir+"/pica.sh")

		// chroot
		if ret, msg, err := ChrootExecShell(ConfigInfo.Rootfsdir, ConfigInfo.DebWorkdir+"/pica.sh", []string{ConfigInfo.DebWorkdir}); !ret {
			logger.Fatal("chroot exec shell failed:", msg, err)
			return
		}

		// copy deb data
		// fixme(heysion): todo

		// fixme(jianqiang)
		// make new directory that need to be created for linglong files stucturesk
		// 定义拷贝的目标目录
		ConfigInfo.ExportDir = ConfigInfo.Workdir + "/export/runtime"
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
		}
		logger.Debugf("exclude so list:", excludeSoList)

		//binReactor.FixElfLDDPath(binReactor.SearchPath + "bin/lib")
		//
		// GetFindElfMissDepends(ConfigInfo.Basedir + "/lib")
		elfLDDLog := ConfigInfo.DebWorkdir + "/elfldd.log"
		elfLDDShell := ConfigInfo.DebWorkdir + "/elfldd.sh"

		logger.Debugf("out: %s , sh: %s", elfLDDLog, elfLDDShell)

		binReactor.RenderElfWithLDD(elfLDDLog, elfLDDShell)

		// // mount shell to chroot
		// logger.Debug("set output in chroot: elfldd.log")
		// if _, msg, err := ExecAndWait(10, "mount", "-B", ConfigInfo.DebWorkdir+"/elfldd.log", ConfigInfo.Rootfsdir+ConfigInfo.DebWorkdir+"/elfldd.log"); err != nil {
		// 	logger.Fatalf("mount %s to %s failed! ", ConfigInfo.Rootfsdir+ConfigInfo.DebWorkdir+"/elfldd.log", err, msg)
		// }

		// chroot
		if ret, msg, err := ChrootExecShell(ConfigInfo.Rootfsdir, elfLDDShell, []string{ConfigInfo.FilesSearchPath}); !ret {
			logger.Fatal("chroot exec shell failed:", msg, err)
			return
		}

		// check result with chroot exec shell
		if ret, err := CheckFileExits(elfLDDLog); !ret {
			logger.Fatal("chroot exec shell failed:", ret, err)
			return
		}

		// read elfldd.log
		logger.Debug("read elfldd.log", elfLDDLog)
		if elfLDDLogFile, err := os.Open(elfLDDLog); err != nil {
			logger.Fatal("open elfldd.log failed:", err)
			//elfLDDLogFile.Close()
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
			// for _, exfind := range excludeSoList {
			// 	if len(exfind) > 0 {
			// 		deleteKeyList := FilterMap(binReactor.ElfNeedPath, func(str string) bool {
			// 			return strings.HasSuffix(str, exfind)
			// 		})

			// 		if len(deleteKeyList) > 0 {
			// 			for _, v := range deleteKeyList {
			// 				delete(binReactor.ElfNeedPath, v)
			// 			}
			// 		}
			// 	}
			// }
			logger.Debugf("fix exclude so list: %v", binReactor.ElfNeedPath)

		}

		logger.Debugf("found %d elf need objects", len(binReactor.ElfNeedPath))

		binReactor.CopyElfNeedPath(ConfigInfo.Rootfsdir, ConfigInfo.FilesSearchPath)

		// fix library
		// if msg, ret := GetElfNeedWithLDD("/bin/bash"); ret != nil {
		// 	logger.Debug("get elf need failed: ", msg)
		// }

		// fix desktop
		// FixDesktop()
		ConfigInfo.FixDesktop(DebConf.Info.Appid)

		// update info.json
		CreateInfo(ConfigInfo.ExportDir, DebConf)

		// umount

		// build uab
		// ll-builder export --local

	},
	PostRun: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Inside rootCmd PostRun with args: %v\n", args)
		//ConfigInfo.MountsItem.DoUmountAOnce()

		// todo(when  umount?)
		// // umount overlayfs
		// logger.Debug("umount rootfs")
		// if ret, err := rfs.UmountRfs(ConfigInfo.Rootfsdir); !ret {
		// 	logger.Error("mount iso failed!", err)
		// }

		// // umount squashfs
		// logger.Debug("umount squashfs")
		// if ret, err := rfs.UmountSquashfs(ConfigInfo.Workdir + "/iso/live"); !ret {
		// 	logger.Error("mount iso failed!", err)
		// }

		// // umount iso
		// logger.Debug("umount iso")
		// if ret, err := rfs.UmountIso(ConfigInfo.Workdir + "/iso/mount"); !ret {
		// 	logger.Error("mount iso failed!", err)
		// }
		// //  umount ConfigInfo.Rootfsdir
		// logger.Debug("umount mounts devs")
		// ConfigInfo.MountsItem.DoUmountAOnce()

	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Inside rootCmd PersistentPostRun with args: %v\n", args)
	},
}

// rootCmd represents the convert command
var rootCmd = &cobra.Command{
	Use:   "ll-pica",
	Short: "debian package convert linglong package",
	Long: `Convert the deb to uab. For example:
Simple:
	ll-pica init 
	ll-pica convert abc.deb --config config.yaml --cache-file=/mnt/workdir/cache.yaml
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

func main() {

	// logger, _ := zap.NewProduction()
	logger = InitLog()
	defer logger.Sync()

	// init cmd add
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&ConfigInfo.Config, "config", "c", "", "config")
	initCmd.Flags().StringVarP(&ConfigInfo.Workdir, "workdir", "w", "", "work directory")
	//initCmd.Flags().StringVarP(&ConfigInfo.iso, "iso", "f", "", "iso")
	initCmd.Flags().BoolVarP(&ConfigInfo.Cache, "keep-cached", "k", true, "keep cached")
	err := initCmd.MarkFlagRequired("config")
	if err != nil {
		logger.Fatal("config required failed", err)
		return
	}

	// convert cmd add
	rootCmd.AddCommand(convertCmd)
	convertCmd.Flags().StringVarP(&TransInfo.Yamlconfig, "config", "c", "", "config")
	convertCmd.Flags().StringVarP(&TransInfo.Workdir, "workdir", "w", "", "work directory")
	convertCmd.Flags().StringVarP(&TransInfo.CachePath, "cache-file", "f", "", "cache yaml file")
	convertCmd.Flags().StringVarP(&TransInfo.DebPath, "deb-file", "d", "", "deb file")

	err = convertCmd.MarkFlagRequired("config")
	if err != nil {
		logger.Fatal("yaml config required failed", err)
	}
	// err = convertCmd.MarkFlagRequired("deb-file")
	// if err != nil {
	// 	logger.Fatal("deb file required failed", err)
	// }

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// root cmd add
	rootCmd.PersistentFlags().BoolVarP(&ConfigInfo.Verbose, "verbose", "v", false, "verbose output")

	if ConfigInfo.Workdir == "" {
		ConfigInfo.Workdir = "/mnt/workdir"
	}

	if TransInfo.Workdir == "" {
		TransInfo.Workdir = "/mnt/workdir"
	}
	// fix cache path
	if TransInfo.CachePath == "" {
		TransInfo.CachePath = TransInfo.Workdir + "/cache.yaml"
	}

	ConfigInfo.MountsItem.Mounts = make(map[string]MountItem)

	err = rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
