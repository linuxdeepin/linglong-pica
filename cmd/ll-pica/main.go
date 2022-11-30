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

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"pkg.deepin.com/linglong/pica/cmd/ll-pica/core"
	"pkg.deepin.com/linglong/pica/cmd/ll-pica/core/comm"
	"pkg.deepin.com/linglong/pica/cmd/ll-pica/core/info"
	"pkg.deepin.com/linglong/pica/cmd/ll-pica/core/linglong"
	"pkg.deepin.com/linglong/pica/cmd/ll-pica/utils/fs"
	"pkg.deepin.com/linglong/pica/cmd/ll-pica/utils/log"
	"pkg.deepin.com/linglong/pica/cmd/ll-pica/utils/rfs"
)

var disableDevelop string
var SdkConf comm.BaseConfig

func SetOverlayfs(lower string, upper string, workdir string) error {
	log.Logger.Debug("SetOverlayfs :", lower, upper, workdir)
	// mount lower dir to upper dir
	// mount -t overlay overlay -o lowerdir=$WORK_DIR/lower,upperdir=$WORK_DIR/upper,workdir=$WORK_DIR/work $WORK_DIR/merged
	tempDir := comm.ConfigInfo.Workdir + "/temp"
	err := os.Mkdir(tempDir, 0755)
	if os.IsNotExist(err) {
		log.Logger.Error("mkdir failed: ", err)
		return err
	}
	msg := fmt.Sprintf("lowerdir=%s:%s,upperdir=%s,workdir=%s", lower, upper, workdir, tempDir)
	_, msg, err = comm.ExecAndWait(10, "mount", "-t", "overlay", "overlay", "-o", msg, comm.ConfigInfo.Rootfsdir)
	if err != nil {
		log.Logger.Error("mount overlayfs failed: ", msg, err)
	}
	return nil
}

func UmountOverlayfs(workdir string) error {
	log.Logger.Debug("UmountOverlayfs :", workdir)
	// umount upper dir
	_, msg, err := comm.ExecAndWait(10, "umount", workdir)
	if err != nil {
		log.Logger.Error("umount overlayfs failed: ", msg, err)
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
		if configPath, err := filepath.Abs(comm.ConfigInfo.Config); err != nil {
			log.Logger.Errorf("Trans %s err: %s ", comm.ConfigInfo.Config, err)
		} else {
			comm.ConfigInfo.Config = configPath
		}

		if workPath, err := filepath.Abs(comm.ConfigInfo.Workdir); err != nil {
			log.Logger.Errorf("Trans %s err: %s ", comm.ConfigInfo.Workdir, err)
		} else {
			comm.ConfigInfo.Workdir = workPath
		}

		log.Logger.Debug("begin process cache: ", comm.ConfigInfo.Cache)
		configCache := comm.ConfigInfo.Workdir + "/cache.yaml"
		runtimeDir := comm.ConfigInfo.Workdir + "/runtime"
		isoDir := comm.ConfigInfo.Workdir + "/iso"

		ClearRuntime := func() {
			log.Logger.Debug("begin clear runtime")
			if _, err := os.Stat(runtimeDir); !os.IsNotExist(err) {
				log.Logger.Debugf("remove runtime: %s", runtimeDir)
				err = os.RemoveAll(runtimeDir)
				if err != nil {
					log.Logger.Errorf("remove error", err)
					return
				}
			}
		}

		if _, err := os.Stat(configCache); !os.IsNotExist(err) && comm.ConfigInfo.Cache {
			// load cache.yaml
			log.Logger.Debugf("load: %s", configCache)
			cacheFd, err := ioutil.ReadFile(configCache)
			if err != nil {
				log.Logger.Warnf("read error: %s", err)
				return
			}

			err = yaml.Unmarshal(cacheFd, &comm.ConfigInfo)
			if err != nil {
				log.Logger.Warnf("unmarshal error: %s", err)
				return
			}
			log.Logger.Debugf("load cache.yaml success: %s", configCache)

			log.Logger.Debug("clear runtime: ", comm.ConfigInfo.IsRuntimeFetch)
			if !comm.ConfigInfo.IsRuntimeFetch {
				// fixme:(heysion) double fetch with ostree cached
				ClearRuntime()
			}

			err = os.Mkdir(runtimeDir, 0755)
			if err != nil {
				log.Logger.Info("create runtime dir error: ", err)
			}

			err = os.Mkdir(isoDir, 0755)
			if err != nil {
				log.Logger.Warn("create iso dir error: ", err)
			}

			return // Config Cache exist
		} else {
			log.Logger.Debug("Config Cache not exist")
			if !comm.ConfigInfo.IsRuntimeCheckout {
				err := os.RemoveAll(comm.ConfigInfo.RuntimeBasedir)
				if err != nil {
					log.Logger.Errorf("remove error", err)
				}
			}
			return // Config Cache not exist
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Logger.Debug("begin run: ", comm.ConfigInfo.Config)

		if comm.ConfigInfo.Config != "" {
			yamlFile, err := ioutil.ReadFile(comm.ConfigInfo.Config)
			if err != nil {
				log.Logger.Errorf("get %s error: %v", comm.ConfigInfo.Config, err)
				return
			}

			err = yaml.Unmarshal(yamlFile, &SdkConf)
			if err != nil {
				log.Logger.Errorf("error: %v", err)
				return
			}
		}

		for idx, baseInfo := range SdkConf.SdkInfo.Base {
			log.Logger.Debugf("get %d %s", idx, baseInfo)

			switch baseInfo.Type {
			case "ostree":
				if !comm.ConfigInfo.IsRuntimeFetch {
					log.Logger.Debugf("ostree init %s", comm.ConfigInfo.IsRuntimeFetch)

					comm.ConfigInfo.RuntimeOstreeDir = comm.ConfigInfo.Workdir + "/runtime"
					if ret := SdkConf.SdkInfo.Base[idx].InitOstree(comm.ConfigInfo.RuntimeOstreeDir); !ret {
						log.Logger.Warn("init ostree failed")
						comm.ConfigInfo.IsRuntimeFetch = false
						continue
					} else {
						comm.ConfigInfo.IsRuntimeFetch = true
					}

					comm.ConfigInfo.RuntimeBasedir = comm.ConfigInfo.Workdir + "/runtimedir"
					if ret := SdkConf.SdkInfo.Base[idx].CheckoutOstree(comm.ConfigInfo.RuntimeBasedir); !ret {
						log.Logger.Warn("checkout ostree failed")
						comm.ConfigInfo.IsRuntimeCheckout = false
						continue
					} else {
						comm.ConfigInfo.IsRuntimeCheckout = true
					}
				}
				continue
			case "iso":
				if !comm.ConfigInfo.IsIsoDownload {
					log.Logger.Debugf("iso download %s", comm.ConfigInfo.IsIsoDownload)

					comm.ConfigInfo.IsoPath = comm.ConfigInfo.Workdir + "/iso/base.iso"
					ret, err := fs.CheckFileExits(comm.ConfigInfo.IsoPath)
					if err != nil {
						log.Logger.Debugf("%v not exists, err: %v", comm.ConfigInfo.IsoPath, err)
					}
					if ret {
						SdkConf.SdkInfo.Base[idx].Path = comm.ConfigInfo.IsoPath
						if ret := SdkConf.SdkInfo.Base[idx].CheckIsoHash(); !ret {
							comm.ConfigInfo.IsIsoChecked = false
							fs.RemovePath(comm.ConfigInfo.IsoPath)
							SdkConf.SdkInfo.Base[idx].Path = ""
						} else {
							log.Logger.Debugf("download skipped because of %s cached", comm.ConfigInfo.IsoPath)
							comm.ConfigInfo.IsIsoChecked = true
							continue
						}
					}

					if ret := SdkConf.SdkInfo.Base[idx].FetchIsoFile(comm.ConfigInfo.Workdir, comm.ConfigInfo.IsoPath); !ret {
						comm.ConfigInfo.IsIsoDownload = false
						log.Logger.Errorf("download iso failed")
						return
					} else {
						comm.ConfigInfo.IsIsoDownload = true
					}
					log.Logger.Debug("iso download success")
				}

				if !comm.ConfigInfo.IsIsoChecked {
					log.Logger.Debug("iso check hash")
					if ret := SdkConf.SdkInfo.Base[idx].CheckIsoHash(); !ret {
						comm.ConfigInfo.IsIsoChecked = false
						log.Logger.Errorf("check iso hash failed")
						return
					} else {
						comm.ConfigInfo.IsIsoChecked = true
					}
					log.Logger.Debug("iso check hash success")
				}
				continue
			}
		}

		comm.ConfigInfo.Initdir = comm.ConfigInfo.Workdir + "/initdir"
		log.Logger.Debug("set initdir: ", comm.ConfigInfo.Initdir)

		// 不读取缓存文件时，需清理initdir
		ret, err := fs.CheckFileExits(comm.ConfigInfo.Initdir)
		if ret && err == nil && !comm.ConfigInfo.IsInited {
			ret, err = fs.RemovePath(comm.ConfigInfo.Initdir)
			if !ret || err != nil {
				log.Logger.Errorf("failed to remove %s\n", comm.ConfigInfo.Initdir)
			}
			ret, err = fs.CreateDir(comm.ConfigInfo.Initdir)
			if !ret || err != nil {
				log.Logger.Errorf("failed to create %s\n", comm.ConfigInfo.Initdir)
			}
		} else {
			ret, err = fs.CreateDir(comm.ConfigInfo.Initdir)
			if !ret || err != nil {
				log.Logger.Errorf("failed to create %s\n", comm.ConfigInfo.Initdir)
			}
		}

		log.Logger.Debug("end init")
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		log.Logger.Debug("begin mount iso: ", comm.ConfigInfo.IsoPath)

		comm.ConfigInfo.IsoMountDir = comm.ConfigInfo.Workdir + "/iso/mount"
		err := errors.New("")
		if ret, err := fs.CheckFileExits(comm.ConfigInfo.IsoMountDir); err != nil && !ret {
			err = os.Mkdir(comm.ConfigInfo.IsoMountDir, 0755)
			if os.IsNotExist(err) {
				log.Logger.Error("mkdir iso mount dir failed!", err)
			}
		}

		var msg string
		_, msg, err = comm.ExecAndWait(10, "mount", "-o", "loop", comm.ConfigInfo.IsoPath, comm.ConfigInfo.IsoMountDir)
		if err != nil {
			log.Logger.Warnf("mount iso failed!", msg, err)
		}
		UmountIsoDir := func() {
			comm.ExecAndWait(10, "umount", comm.ConfigInfo.IsoMountDir)
		}

		defer UmountIsoDir()

		baseDir := comm.ConfigInfo.Workdir + "/iso/live"
		err = os.Mkdir(baseDir, 0755)
		if os.IsNotExist(err) {
			log.Logger.Error("mkdir iso mount dir failed!", err)
		}

		_, msg, err = comm.ExecAndWait(10, "mount", comm.ConfigInfo.IsoMountDir+"/live/filesystem.squashfs", baseDir)
		if err != nil {
			log.Logger.Error("mount squashfs failed!", msg, err)
		}

		UmountSquashfsDir := func() {
			comm.ExecAndWait(10, "umount", baseDir)
		}
		defer UmountSquashfsDir()

		comm.ConfigInfo.Rootfsdir = comm.ConfigInfo.Workdir + "/rootfs"
		err = os.Mkdir(comm.ConfigInfo.Rootfsdir, 0755)
		if os.IsNotExist(err) {
			log.Logger.Error("mkdir rootfsdir failed!", err)
		}

		// mount overlay to base dir
		if ret, err := fs.CheckFileExits(comm.ConfigInfo.RuntimeBasedir + "/files"); err == nil && ret {
			SetOverlayfs(baseDir, comm.ConfigInfo.RuntimeBasedir+"/files", comm.ConfigInfo.Initdir)
		} else {
			SetOverlayfs(baseDir, comm.ConfigInfo.RuntimeBasedir, comm.ConfigInfo.Initdir)
		}

		UmountRootfsDir := func() {
			comm.ExecAndWait(10, "umount", comm.ConfigInfo.Rootfsdir)
		}
		defer UmountRootfsDir()

		comm.ConfigInfo.MountsItem.FillMountRules()

		fmt.Printf("Inside rootCmd PostRun with args: %v\n", args)
		comm.ConfigInfo.IsInited = true

		yamlData, err := yaml.Marshal(&comm.ConfigInfo)
		if err != nil {
			log.Logger.Errorf("convert to yaml failed!")
		}

		err = ioutil.WriteFile(fmt.Sprintf("%s/cache.yaml", comm.ConfigInfo.Workdir), yamlData, 0644)
		if err != nil {
			log.Logger.Error("write cache.yaml failed!")
		}

		comm.ConfigInfo.MountsItem.DoMountALL()
		defer comm.ConfigInfo.MountsItem.DoUmountALL()

		// write source.list
		log.Logger.Debugf("Start write sources.list !")
		if ret := SdkConf.SdkInfo.Extra.WriteRootfsRepo(comm.ConfigInfo); !ret {
			log.Logger.Errorf("Write sources.list failed!")
		}

		// write extra shell
		if len(SdkConf.SdkInfo.Extra.Cmd) > 0 {
			SdkConf.SdkInfo.Extra.RenderExtraShell(comm.ConfigInfo.Rootfsdir + "/init.sh")
			defer func() { fs.RemovePath(comm.ConfigInfo.Rootfsdir + "/init.sh") }()
			core.ChrootExecShellBare(comm.ConfigInfo.Rootfsdir, comm.ConfigInfo.Rootfsdir+"/init.sh")
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
	ll-pica convert  --config config.yaml --workdir=/mnt/workdir
	`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.Logger.Debugf("workdir %s ,cache file %s", comm.TransInfo.Workdir, comm.TransInfo.CachePath)

		if comm.TransInfo.CachePath == "" {
			if ret, err := fs.CheckFileExits(comm.TransInfo.Workdir + "/cache.yaml"); !ret {
				log.Logger.Fatal("cache-file required failed", err)
				return
			}
			comm.TransInfo.CachePath = comm.TransInfo.Workdir + "/cache.yaml"
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if comm.ConfigInfo.Verbose {
			log.Logger.Info("verbose mode enabled")
			comm.TransInfo.Verbose = true
		}

		// 转换获取路径为绝对路径
		if yamlPath, err := filepath.Abs(comm.TransInfo.Yamlconfig); err != nil {
			log.Logger.Errorf("Trans %s err: %s ", comm.TransInfo.Yamlconfig, err)
		} else {
			comm.TransInfo.Yamlconfig = yamlPath
		}

		if workPath, err := filepath.Abs(comm.TransInfo.Workdir); err != nil {
			log.Logger.Errorf("Trans %s err: %s ", comm.TransInfo.Workdir, err)
		} else {
			comm.TransInfo.Workdir = workPath
		}

		if cachePath, err := filepath.Abs(comm.TransInfo.CachePath); err != nil {
			log.Logger.Errorf("Trans %s err: %s ", comm.TransInfo.CachePath, err)
		} else {
			comm.TransInfo.CachePath = cachePath
		}

		// 修复CachePath参数
		if ret, err := comm.TransInfo.FixCachePath(); !ret || err != nil {
			log.Logger.Fatal("can not found: ", comm.TransInfo.Workdir)
		}

		log.Logger.Debug("load yaml config", comm.TransInfo.Yamlconfig)
		if ret, err := fs.CheckFileExits(comm.TransInfo.Yamlconfig); err != nil && !ret {
			log.Logger.Fatal("can not found: ", err)
		} else {
			log.Logger.Debugf("load: %s", comm.TransInfo.Yamlconfig)
			cacheFd, err := ioutil.ReadFile(comm.TransInfo.Yamlconfig)
			if err != nil {
				log.Logger.Fatalf("read error: %s %s", err, err)
				return
			}
			err = yaml.Unmarshal(cacheFd, &comm.DebConf)
			if err != nil {
				log.Logger.Fatalf("unmarshal error: %s", err)
				return
			}
		}

		log.Logger.Debug("load cache.yaml", comm.TransInfo.CachePath)
		if ret, err := fs.CheckFileExits(comm.TransInfo.CachePath); err == nil && ret {
			// load cache.yaml
			log.Logger.Debugf("load cache: %s", comm.TransInfo.CachePath)
			cacheFd, err := ioutil.ReadFile(comm.TransInfo.CachePath)
			if err != nil {
				log.Logger.Warnf("read error: %s %s", err, err)
				return
			}
			err = yaml.Unmarshal(cacheFd, &comm.ConfigInfo)
			if err != nil {
				log.Logger.Warnf("unmarshal error: %s", err)
				return
			}
		} else {
			log.Logger.Fatalf("can not found: %s", err)
			return
		}

		comm.ConfigInfo.DebPath = comm.TransInfo.DebPath
		comm.ConfigInfo.Yamlconfig = comm.TransInfo.Yamlconfig
		comm.ConfigInfo.Verbose = comm.TransInfo.Verbose
		comm.ConfigInfo.CachePath = comm.TransInfo.CachePath
		comm.ConfigInfo.DebugMode = comm.TransInfo.DebugMode

		// 创建debdir
		comm.ConfigInfo.DebWorkdir = comm.ConfigInfo.Workdir + "/debdir"
		if ret, err := fs.CheckFileExits(comm.ConfigInfo.DebWorkdir); !ret && err != nil {
			ret, err = fs.CreateDir(comm.ConfigInfo.DebWorkdir)
			if !ret || err != nil {
				log.Logger.Errorf("failed to create %s\n", comm.ConfigInfo.DebWorkdir)
			}
		}

		// 新建basedir
		comm.ConfigInfo.Basedir = comm.ConfigInfo.Workdir + "/basedir"
		if ret, err := fs.CheckFileExits(comm.ConfigInfo.Basedir); ret && err == nil {
			ret, err = fs.RemovePath(comm.ConfigInfo.Basedir)
			if !ret || err != nil {
				log.Logger.Errorf("failed to remove %s\n", comm.ConfigInfo.Basedir)
			}
			ret, err = fs.CreateDir(comm.ConfigInfo.Basedir)
			if !ret || err != nil {
				log.Logger.Errorf("failed to create %s\n", comm.ConfigInfo.Basedir)
			}
		} else {
			ret, err = fs.CreateDir(comm.ConfigInfo.Basedir)
			if !ret || err != nil {
				log.Logger.Errorf("failed to create %s\n", comm.ConfigInfo.Basedir)
			}
		}

		if ret, err := rfs.MountIso(comm.ConfigInfo.Workdir+"/iso/mount", comm.ConfigInfo.IsoPath); !ret {
			log.Logger.Warnf("mount iso failed!", err)
		}

		if ret, err := rfs.MountSquashfs(comm.ConfigInfo.Workdir+"/iso/live", comm.ConfigInfo.Workdir+"/iso/mount/live/filesystem.squashfs"); !ret {
			log.Logger.Warnf("mount live failed!", err)
		}

		// mount overlay to base dir
		log.Logger.Debug("Rootfsdir:", comm.ConfigInfo.Rootfsdir, "runtimeBasedir:", comm.ConfigInfo.RuntimeBasedir, "basedir:", comm.ConfigInfo.Basedir, "workdir:", comm.ConfigInfo.Workdir)

		fs.CreateDir(comm.ConfigInfo.Workdir + "/tmpdir")
		if ret, err := fs.CheckFileExits(comm.ConfigInfo.RuntimeBasedir + "/files"); err == nil && ret {
			if ret, err := rfs.MountRfsWithOverlayfs(comm.ConfigInfo.RuntimeBasedir+"/files", comm.ConfigInfo.Workdir+"/iso/live", comm.ConfigInfo.Initdir, comm.ConfigInfo.Basedir, comm.ConfigInfo.Workdir+"/tmpdir", comm.ConfigInfo.Rootfsdir); !ret {
				log.Logger.Warnf("mount rootfs failed!", err)
			}
		} else {
			if ret, err := rfs.MountRfsWithOverlayfs(comm.ConfigInfo.RuntimeBasedir, comm.ConfigInfo.Workdir+"/iso/live", comm.ConfigInfo.Initdir, comm.ConfigInfo.Basedir, comm.ConfigInfo.Workdir+"/tmpdir", comm.ConfigInfo.Rootfsdir); !ret {
				log.Logger.Warnf("mount rootfs failed!", err)
			}
		}

		comm.ConfigInfo.MountsItem.DoMountALL()
	},
	Run: func(cmd *cobra.Command, args []string) {
		// fetch deb file
		// DebConfig
		log.Logger.Debugf("debConfig deb:%v", comm.DebConf.FileElement.Deb)
		for idx, _ := range comm.DebConf.FileElement.Deb {
			// fetch deb file
			if len(comm.DebConf.FileElement.Deb[idx].Ref) > 0 {
				// NOTE: work with go1.15 but feature not sure .
				debFilePath := comm.ConfigInfo.DebWorkdir + "/" + filepath.Base(comm.DebConf.FileElement.Deb[idx].Ref)
				log.Logger.Warnf("deb file :%s", debFilePath)
				if ret, err := fs.CheckFileExits(debFilePath); err == nil && ret {
					comm.DebConf.FileElement.Deb[idx].Path = debFilePath
					if ret := comm.DebConf.FileElement.Deb[idx].CheckDebHash(); ret {
						log.Logger.Infof("download skipped because of %s cached", debFilePath)
						continue
					} else {
						fs.RemovePath(debFilePath)
						comm.DebConf.FileElement.Deb[idx].Path = ""
					}
				}
				// fetch deb file
				comm.DebConf.FileElement.Deb[idx].FetchDebFile(debFilePath)
				log.Logger.Debugf("fetch deb path:[%d] %s", idx, debFilePath)
				// check deb hash
				if ret := comm.DebConf.FileElement.Deb[idx].CheckDebHash(); !ret {
					log.Logger.Warnf("check deb hash failed! : ", comm.DebConf.FileElement.Deb[idx].Name)
					continue
				}
				log.Logger.Info("download %s success.", comm.DebConf.FileElement.Deb[idx].Path)
			}
			continue
		}

		// render DebConfig to template save to pica.sh
		// clear pica.sh cache
		picaShellPath := comm.ConfigInfo.DebWorkdir + "/pica.sh"
		if ret, err := fs.CheckFileExits(picaShellPath); err == nil && ret {
			fs.RemovePath(picaShellPath)
		}

		log.Logger.Infof("render %s script.", picaShellPath)
		core.RenderDebConfig(comm.DebConf, picaShellPath)

		// exec script in chroot
		log.Logger.Info("exec script in chroot")
		if ret, msg, err := core.ChrootExecShell(comm.ConfigInfo.Rootfsdir, picaShellPath, []string{comm.ConfigInfo.DebWorkdir}); !ret {
			log.Logger.Fatal("exec pica script in chroot failed! :", msg, err)
			return
		} else {
			// 打印详细日志时输出
			comm.LoggerVerbose("exec pica script in chroot output: %s", msg)
		}

		// 定义拷贝的目标目录
		comm.ConfigInfo.ExportDir = comm.ConfigInfo.Workdir + "/" + comm.DebConf.Info.Appid + "/export/runtime"
		// 导出export目录
		comm.ConfigInfo.Export()

		var binReactor core.BinFormatReactor

		// set files directory
		binReactor.SearchPath = comm.ConfigInfo.FilesSearchPath

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
		log.Logger.Debugf("exclude so list:", excludeSoList)

		// check  dlopen if it exists append depends to list
		log.Logger.Debug("call GetEntryDlopenList:")
		binReactor.GetEntryDlopenList(excludeSoList)
		log.Logger.Debug("call GetEntryDlopenList: %v", binReactor.ElfEntrySoPath)

		elfLDDLog := comm.ConfigInfo.DebWorkdir + "/elfldd.log"
		elfLDDShell := comm.ConfigInfo.DebWorkdir + "/elfldd.sh"

		// clear history
		if ret, err := fs.CheckFileExits(elfLDDLog); err == nil && ret {
			fs.RemovePath(elfLDDLog)
		}

		if ret, err := fs.CheckFileExits(elfLDDShell); err == nil && ret {
			fs.RemovePath(elfLDDShell)
		}

		log.Logger.Debugf("out: %s , sh: %s", elfLDDLog, elfLDDShell)

		binReactor.RenderElfWithLDD(elfLDDLog, elfLDDShell)

		// chroot
		if ret, msg, err := core.ChrootExecShell(comm.ConfigInfo.Rootfsdir, elfLDDShell, []string{comm.ConfigInfo.FilesSearchPath}); !ret {
			log.Logger.Fatal("chroot exec shell failed:", msg, err)
			return
		}

		// check result with chroot exec shell
		if ret, err := fs.CheckFileExits(elfLDDLog); !ret {
			log.Logger.Fatal("chroot exec shell failed:", ret, err)
			return
		}

		// read elfldd.log
		log.Logger.Debug("read elfldd.log", elfLDDLog)
		if elfLDDLogFile, err := os.Open(elfLDDLog); err != nil {
			log.Logger.Fatal("open elfldd.log failed:", err)
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
			log.Logger.Debugf("fix exclude so list: %v", binReactor.ElfNeedPath)

		}
		log.Logger.Debugf("found %d elf need objects", len(binReactor.ElfNeedPath))
		binReactor.CopyElfNeedPath(comm.ConfigInfo.Rootfsdir, comm.ConfigInfo.FilesSearchPath)

		builder := linglong.LinglongBuder{
			Appid:       comm.DebConf.Info.Appid,
			Version:     comm.DebConf.Info.Version,
			Description: comm.DebConf.Info.Description,
			Runtime:     "org.deepin.Runtime",
			Rversion:    "",
		}

		// load runtime.json
		log.Logger.Debugf("loader runtimedir %s", comm.ConfigInfo.RuntimeBasedir)
		builder.LoadRuntimeInfo(comm.ConfigInfo.RuntimeBasedir + "/info.json")

		// run.sh
		// fixme: 依据kind 字段生成 run.sh 的模板

		// FixDesktop()
		comm.ConfigInfo.FixDesktop(comm.DebConf.Info.Appid)

		// update info.json
		info.CreateInfo(comm.ConfigInfo, &comm.DebConf, builder)

		// 修正版本号
		builder.Version = comm.DebConf.Info.Version

		log.Logger.Debugf("update linglong builder: %v", builder)

		// create linglong.yaml
		builder.CreateLinglongYamlBuilder(comm.ConfigInfo.ExportDir + "/linglong.yaml")
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Inside rootCmd PostRun with args: %v\n", args)
		comm.ConfigInfo.MountsItem.DoUmountALL()

		// umount overlayfs
		log.Logger.Debug("umount rootfs")
		if ret, err := rfs.UmountRfs(comm.ConfigInfo.Rootfsdir); !ret {
			log.Logger.Warnf("umount rootfs failed!", err)
		}

		// umount squashfs
		log.Logger.Debug("umount squashfs")
		if ret, err := rfs.UmountSquashfs(comm.ConfigInfo.Workdir + "/iso/live"); !ret {
			log.Logger.Warnf("umount squashfs failed!", err)
		}

		// umount iso
		log.Logger.Debug("umount iso")
		if ret, err := rfs.UmountIso(comm.ConfigInfo.Workdir + "/iso/mount"); !ret {
			log.Logger.Warnf("umount iso failed!", err)
		}
	},
}

var rootCmd = &cobra.Command{
	Use:   "ll-pica",
	Short: "debian package convert linglong package",
	Long: `Convert the deb to uab. For example:
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
		log.Logger.Infof("parse input app:", comm.ConfigInfo.AppId)

		// 转化工作目录为绝对路径
		if workPath, err := filepath.Abs(comm.ConfigInfo.Workdir); err != nil {
			log.Logger.Errorf("Trans %s err: %s ", comm.ConfigInfo.Workdir, err)
		} else {
			comm.ConfigInfo.Workdir = workPath
		}

		// auth username
		if comm.ConfigInfo.AppUsername == "" || comm.ConfigInfo.AppPasswords == "" {
			comm.ConfigInfo.AppAuthType = comm.AppLoginWithKeyfile
		} else {
			log.Logger.Infof("app login with password")
			comm.ConfigInfo.AppAuthType = comm.AppLoginWithPassword
		}

		// AppKeyFile path
		comm.ConfigInfo.AppKeyFile = fs.GetHomePath() + "/.linglong/.user.json"
		// keyfile
		if ret, err := fs.CheckFileExits(comm.ConfigInfo.AppKeyFile); err != nil && !ret && (comm.ConfigInfo.AppAuthType == comm.AppLoginWithKeyfile) {
			log.Logger.Errorf("not found keyfile %v, please push with user and password!", comm.ConfigInfo.AppKeyFile)
			comm.ConfigInfo.AppAuthType = comm.AppLoginFailed
			return
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Logger.Infof("app path %v", comm.ConfigInfo.Workdir+"/"+comm.ConfigInfo.AppId+"/export/runtime")
		appDataPath := comm.ConfigInfo.Workdir + "/" + comm.ConfigInfo.AppId + "/export/runtime"
		if ret, err := fs.CheckFileExits(appDataPath); err != nil && !ret {
			log.Logger.Errorf("app data dir not exist : %v", appDataPath)
			return
		}

		// 执行上传操作
		// 获取当前路径
		cwdPath, err := os.Getwd()
		if err != nil {
			log.Logger.Errorf("get cwd path Failed %v", err)
			return
		}
		// 进入appDataPath
		err = os.Chdir(appDataPath)
		if err != nil {
			log.Logger.Errorf("chdir failed: %s", err)
			return
		}

		if ret, err := comm.LinglongBuilderWarp(comm.ConfigInfo.AppAuthType, &comm.ConfigInfo); !ret {
			log.Logger.Errorf("%v push failed: %v", appDataPath, err)
			return
		}

		// 退出appDatapath
		err = os.Chdir(cwdPath)
		if err != nil {
			log.Logger.Errorf("chdir failed: %s", err)
			return
		}
	},
	PostRun: func(cmd *cobra.Command, args []string) {

	},
}

func main() {
	log.Logger = log.InitLog()
	defer log.Logger.Sync()

	rootCmd.AddCommand(initCmd)
	rootCmd.PersistentFlags().BoolVarP(&comm.ConfigInfo.Verbose, "verbose", "v", false, "verbose output")
	initCmd.Flags().StringVarP(&comm.ConfigInfo.Config, "config", "c", "", "config")
	initCmd.Flags().StringVarP(&comm.ConfigInfo.Workdir, "workdir", "w", "", "work directory")
	err := initCmd.MarkFlagRequired("config")
	if err != nil {
		log.Logger.Fatal("config required failed", err)
		return
	}

	rootCmd.AddCommand(convertCmd)
	convertCmd.Flags().StringVarP(&comm.TransInfo.Yamlconfig, "config", "c", "", "config")
	convertCmd.Flags().StringVarP(&comm.TransInfo.Workdir, "workdir", "w", "", "work directory")
	err = convertCmd.MarkFlagRequired("config")
	if err != nil {
		log.Logger.Fatal("yaml config required failed", err)
	}

	if err := convertCmd.MarkFlagRequired("workdir"); err != nil {
		log.Logger.Fatal("workdir required failed", err)
		return
	}

	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringVarP(&comm.ConfigInfo.AppUsername, "username", "u", "", "username")
	pushCmd.Flags().StringVarP(&comm.ConfigInfo.AppPasswords, "passwords", "p", "", "passwords")
	pushCmd.Flags().StringVarP(&comm.ConfigInfo.AppId, "appid", "i", "", "app id")
	pushCmd.Flags().StringVarP(&comm.ConfigInfo.AppChannel, "channel", "c", "linglong", "app channel")
	pushCmd.Flags().StringVarP(&comm.ConfigInfo.AppRepoUrl, "repo", "r", "", "repo url")
	pushCmd.Flags().StringVarP(&comm.ConfigInfo.AppRepoName, "reponame", "n", "", "repo name")
	pushCmd.Flags().StringVarP(&comm.ConfigInfo.Workdir, "workdir", "w", "", "work directory")
	if err := pushCmd.MarkFlagRequired("workdir"); err != nil {
		log.Logger.Fatal("workdir required failed", err)
		return
	}

	if err := pushCmd.MarkFlagRequired("appid"); err != nil {
		log.Logger.Fatal("appid required failed", err)
		return
	}

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// go build -ldflags '-X pkg.deepin.com/linglong/pica/cmd/ll-pica/utils/log.disableLogDebug=yes -X main.disableDevelop=yes'
	// fmt.Printf("disableDevelop: %s\n", disableDevelop)
	if disableDevelop != "" {
		log.Logger.Debugf("develop mode disable")
		comm.TransInfo.DebugMode = false
		comm.ConfigInfo.DebugMode = false
	} else {
		log.Logger.Debugf("develop mode enabled")
		comm.TransInfo.DebugMode = true
		comm.ConfigInfo.DebugMode = true
		// debug mode enable verbose mode
		comm.TransInfo.Verbose = true
		comm.ConfigInfo.Verbose = true
	}

	comm.ConfigInfo.MountsItem.Mounts = make(map[string]comm.MountItem)

	err = rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
