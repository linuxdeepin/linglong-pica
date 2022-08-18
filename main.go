/*
Copyright © 2022 Heysion Y heysion@deepin.com

ts program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

ts program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with ts program. If not, see <http://www.gnu.org/licenses/>.
*/
package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	. "ll-pica/core"
	. "ll-pica/core/comm"
	. "ll-pica/utils/fs"
	. "ll-pica/utils/log"
	"ll-pica/utils/rfs"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// app config with runtime
type Config struct {
	Verbose           bool   `yaml:"verbose"`
	Config            string `yaml:"config"`
	Workdir           string `yaml:"workdir"`
	Basedir           string `yaml:"basedir"`
	IsInited          bool   `yaml:"inited"`
	Cache             bool   `yaml:"cache"`
	CachePath         string `yaml:"cache-path"`
	DebWorkdir        string `yaml:"debdir"`
	debPath           string
	IsRuntimeFetch    bool   `yaml:"runtime-fetched"`
	IsRuntimeCheckout bool   `yaml:"runtime-checkedout"`
	RuntimeBasedir    string `yaml:"runtime-basedir"`
	IsIsoDownload     bool   `yaml:"iso-downloaded"`
	IsoPath           string `yaml:"iso-path"`
	IsoMountDir       string `yaml:"iso-mount-dir"`
	IsIsoChecked      bool   `yaml:"iso-checked"`
	Rootfsdir         string `yaml:"rootfsdir"`
	MountsItem        Mounts `yaml:"mounts"`
	yamlconfig        string
}

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

type MountItem struct {
	MountPoint string `yaml:"mountpoint"`
	Source     string `yaml:"source"`
	Type       string `yaml:"type"`
	IsRbind    bool   `yaml:"bind"`
}

type Mounts struct {
	Mounts map[string]MountItem `yaml:"mounts"`
}

var configInfo Config
var transInfo Config
var logger *zap.SugaredLogger

var debConf DebConfig

func (ts Mounts) DoMountALL() []error {

	logger.Debug("mount list: ", len(ts.Mounts))
	var errs []error
	if len(ts.Mounts) == 0 {
		return errs
	}

	var msg string
	var err error

	for _, item := range ts.Mounts {

		logger.Debugf("mount: ", item.MountPoint, item.Source, item.Type, item.IsRbind)
		if IsRbind := item.IsRbind; IsRbind {

			// sudo mount --rbind /tmp/ /mnt/workdir/rootfs/tmp/
			_, msg, err = ExecAndWait(10, "mount", "--rbind", item.Source, item.MountPoint)
			if err != nil {
				logger.Warnf("mount bind failed: ", msg, err)
				errs = append(errs, err)
			}

			// sudo mount --make-rslave /mnt/workdir/rootfs/tmp/
			_, msg, err = ExecAndWait(10, "mount", "--make-rslave", item.MountPoint)
			if err != nil {
				logger.Warnf("mount bind rslave failed: ", msg, err)
				errs = append(errs, err)
			}

		} else {
			_, msg, err = ExecAndWait(10, "mount", "-t", item.Type, item.Source, item.MountPoint)
			if err != nil {
				logger.Warnf("mount failed: ", msg, err)
				errs = append(errs, err)
			}
		}

	}
	return errs
}

func (ts Mounts) DoUmountALL() []error {
	logger.Debug("mount list: ", len(ts.Mounts))
	var errs []error
	if len(ts.Mounts) == 0 {
		return errs
	}

	for _, item := range ts.Mounts {
		logger.Debugf("umount: ", item.MountPoint)
		_, msg, err := ExecAndWait(10, "umount", item.MountPoint)
		if err != nil {
			logger.Warnf("umount failed: ", msg, err)
			errs = append(errs, err)
		} else {
			delete(ts.Mounts, item.MountPoint)
		}

	}
	return errs
}

func (ts Mounts) DoUmountAOnce() []error {
	return nil
	logger.Debug("mount list: ", len(ts.Mounts))
	var errs []error
	if len(ts.Mounts) == 0 {
		return errs
	}

	idx := 0
UMOUNT_ONCE:
	_, msg, err := ExecAndWait(10, "umount", "-R", configInfo.Rootfsdir)
	if err == nil {
		idx++
		if idx < 10 {
			goto UMOUNT_ONCE
		}
	} else {
		logger.Warnf("umount success: ", msg, err)
		errs = append(errs, nil)
	}
	for _, item := range ts.Mounts {
		logger.Debugf("umount: ", item.MountPoint)
		delete(ts.Mounts, item.MountPoint)

	}
	return errs
}

func (ts *Mounts) FillMountRules() {

	logger.Debug("mount list: ", len(ts.Mounts))
	ts.Mounts[configInfo.Rootfsdir+"/dev/"] = MountItem{configInfo.Rootfsdir + "/dev/", "/dev/", "tmpfs", true}
	ts.Mounts[configInfo.Rootfsdir+"/sys/"] = MountItem{configInfo.Rootfsdir + "/sys/", "/sys/", "sysfs", true}
	ts.Mounts[configInfo.Rootfsdir+"/tmp/"] = MountItem{configInfo.Rootfsdir + "/tmp/", "/tmp/", "tmpfs", true}
	ts.Mounts[configInfo.Rootfsdir+"/etc/resolv.conf"] = MountItem{configInfo.Rootfsdir + "/etc/resolv.conf", "/etc/resolv.conf", "tmpfs", true}

	ts.Mounts[configInfo.Rootfsdir+"/proc/"] = MountItem{configInfo.Rootfsdir + "/proc/", "none", "proc", false}

	logger.Debug("mount list: ", len(ts.Mounts))
}

// var RootfsMountList Mounts

var SdkConf BaseConfig

// initCmd represents the init command

func SetOverlayfs(lower string, upper string, workdir string) error {
	logger.Debug("SetOverlayfs :", lower, upper, workdir)
	// mount lower dir to upper dir
	//mount -t overlay overlay -o lowerdir=$WORK_DIR/lower,upperdir=$WORK_DIR/upper,workdir=$WORK_DIR/work $WORK_DIR/merged
	tempDir := configInfo.Workdir + "/temp"
	err := os.Mkdir(tempDir, 0755)
	if os.IsNotExist(err) {
		logger.Error("mkdir failed: ", err)
		return err
	}
	msg := fmt.Sprintf("lowerdir=%s:%s,upperdir=%s,workdir=%s", upper, lower, workdir, tempDir)
	_, msg, err = ExecAndWait(10, "mount", "-t", "overlay", "overlay", "-o", msg, configInfo.Rootfsdir)
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
		logger.Debug("begin process cache: ", configInfo.Cache)
		configCache := fmt.Sprintf("%s/cache.yaml", configInfo.Workdir)
		runtimeDir := fmt.Sprintf("%s/runtime", configInfo.Workdir)
		isoDir := fmt.Sprintf("%s/iso", configInfo.Workdir)

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

		if _, err := os.Stat(configCache); !os.IsNotExist(err) && configInfo.Cache {
			// load cache.yaml
			logger.Debugf("load: %s", configCache)
			cacheFd, err := ioutil.ReadFile(configCache)
			if err != nil {
				logger.Warnf("read error: %s", err)
				return
			}
			err = yaml.Unmarshal(cacheFd, &configInfo)
			if err != nil {
				logger.Warnf("unmarshal error: %s", err)
				return
			}
			logger.Debugf("load cache.yaml success: %s", configCache)

			logger.Debug("clear runtime: ", configInfo.IsRuntimeFetch)
			if !configInfo.IsRuntimeFetch {
				ClearRuntime()
			}

			err = os.Mkdir(runtimeDir, 0755)
			if err != nil {
				logger.Info("create runtime dir error: ", err)
			}

			logger.Debug("clear iso: ", configInfo.IsIsoDownload)
			if !configInfo.IsIsoDownload {
				ClearIso()
			}

			err = os.Mkdir(isoDir, 0755)
			if err != nil {
				logger.Info("create iso dir error: ", err)
			}

			return // Config Cache exist
		} else {
			logger.Debug("Config Cache not exist")
			if !configInfo.IsRuntimeCheckout {
				err := os.RemoveAll(configInfo.RuntimeBasedir)
				if err != nil {
					logger.Errorf("remove error", err)
				}
			}

			return // Config Cache not exist
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debug("begin run: ", configInfo.Config)
		if configInfo.Config != "" {
			yamlFile, err := ioutil.ReadFile(configInfo.Config)
			if err != nil {
				logger.Errorf("get %s error: %v", configInfo.Config, err)
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
				if !configInfo.IsRuntimeFetch {
					logger.Debug("ostree init")
					if ret := SdkConf.SdkInfo.Base[idx].InitOstree(configInfo.Workdir); !ret {
						logger.Error("init ostree failed")
						configInfo.IsRuntimeFetch = false
						return
					} else {
						configInfo.IsRuntimeFetch = true
					}

					configInfo.RuntimeBasedir = fmt.Sprintf("%s/runtimedir", configInfo.Workdir)
					if ret := SdkConf.SdkInfo.Base[idx].CheckoutOstree(configInfo.RuntimeBasedir); !ret {
						logger.Error("checkout ostree failed")
						configInfo.IsRuntimeCheckout = false
						return
					} else {
						configInfo.IsRuntimeCheckout = true
					}

				}
				continue

			case "iso":

				if !configInfo.IsIsoDownload {
					logger.Debug("iso download")

					configInfo.IsoPath = fmt.Sprintf("%s/iso/base.iso", configInfo.Workdir)

					if ret := SdkConf.SdkInfo.Base[idx].FetchIsoFile(configInfo.Workdir, configInfo.IsoPath); !ret {
						configInfo.IsIsoDownload = false
						logger.Errorf("download iso failed")
						return
					} else {
						configInfo.IsIsoDownload = true
					}
					logger.Debug("iso download success")
				}

				if !configInfo.IsIsoChecked {
					logger.Debug("iso check hash")
					if ret := SdkConf.SdkInfo.Base[idx].CheckIsoHash(); !ret {
						configInfo.IsIsoChecked = false
						logger.Errorf("check iso hash failed")
						return
					} else {
						configInfo.IsIsoChecked = true
					}
					logger.Debug("iso check hash success")
				}
				continue
			}
		}

		configInfo.Basedir = fmt.Sprintf("%s/basedir", configInfo.Workdir)
		logger.Debug("set basedir: ", configInfo.Basedir)

		// extra
		logger.Debug("extra init")
		// if SdkConf.SdkInfo.Extra != ExtraInfo{,} {
		// 	logger.Debug(SdkConf.SdkInfo.Extra)
		// }

		logger.Debug("end init")
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		logger.Debugf("config :%+v", configInfo)
		yamlData, err := yaml.Marshal(&configInfo)
		if err != nil {
			logger.Errorf("convert to yaml failed!")
		}
		logger.Debugf("write Config Cache: %v", string(yamlData))
		err = ioutil.WriteFile(fmt.Sprintf("%s/cache.yaml", configInfo.Workdir), yamlData, 0644)
		if err != nil {
			logger.Error("write cache.yaml failed!")
		}

		logger.Debug("begin mount iso: ", configInfo.IsoPath)
		configInfo.IsoMountDir = fmt.Sprintf("%s/iso/mount", configInfo.Workdir)

		if ret, _ := CheckFileExits(configInfo.IsoMountDir); !ret {
			err = os.Mkdir(configInfo.IsoMountDir, 0755)
			if os.IsNotExist(err) {
				logger.Error("mkdir iso mount dir failed!", err)
			}
		}

		var msg string
		_, msg, err = ExecAndWait(10, "mount", "-o", "loop", configInfo.IsoPath, configInfo.IsoMountDir)
		if err != nil {
			logger.Error("mount iso failed!", msg, err)
		}
		UmountIsoDir := func() {
			ExecAndWait(10, "umount", configInfo.IsoMountDir)
		}

		defer UmountIsoDir()

		// mount squashfs to base dir

		baseDir := fmt.Sprintf("%s/iso/live", configInfo.Workdir)
		err = os.Mkdir(baseDir, 0755)
		if os.IsNotExist(err) {
			logger.Error("mkdir iso mount dir failed!", err)
		}
		_, msg, err = ExecAndWait(10, "mount", fmt.Sprintf("%s/live/filesystem.squashfs", configInfo.IsoMountDir), baseDir)
		if err != nil {
			logger.Error("mount squashfs failed!", msg, err)
		}
		UmountSquashfsDir := func() {
			ExecAndWait(10, "umount", baseDir)
		}
		defer UmountSquashfsDir()

		configInfo.Rootfsdir = fmt.Sprintf("%s/rootfs", configInfo.Workdir)
		err = os.Mkdir(configInfo.Rootfsdir, 0755)
		if os.IsNotExist(err) {
			logger.Error("mkdir runtime basedir failed!", err)
		}

		err = os.Mkdir(configInfo.Basedir, 0755)

		if os.IsNotExist(err) {
			logger.Error("mkdir runtime basedir failed!", err)
		}

		// mount overlay to base dir
		SetOverlayfs(baseDir, configInfo.RuntimeBasedir, configInfo.Basedir)

		configInfo.MountsItem.FillMountRules()

		fmt.Printf("Inside rootCmd PostRun with args: %v\n", args)
		configInfo.IsInited = true

		yamlData, err = yaml.Marshal(&configInfo)
		if err != nil {
			logger.Errorf("convert to yaml failed!")
		}
		logger.Debugf("write Config Cache: %v", string(yamlData))
		err = ioutil.WriteFile(fmt.Sprintf("%s/cache.yaml", configInfo.Workdir), yamlData, 0644)
		if err != nil {
			logger.Error("write cache.yaml failed!")
		}

		configInfo.MountsItem.DoMountALL()
		configInfo.MountsItem.DoUmountAOnce()
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
		if transInfo.Verbose {
			fmt.Println(transInfo.Verbose)
		}
		if ret, err := CheckFileExits(transInfo.debPath); err != nil && !ret {
			logger.Fatal("check deb file error:", err)
		}

		logger.Debug("load yaml config", transInfo.yamlconfig)

		if ret, msg := CheckFileExits(transInfo.yamlconfig); !ret {
			logger.Fatal("can not found: ", msg)
		} else {
			logger.Debugf("load: %s", transInfo.yamlconfig)
			cacheFd, err := ioutil.ReadFile(transInfo.yamlconfig)
			if err != nil {
				logger.Fatalf("read error: %s %s", err, msg)
				return
			}
			logger.Debugf("load: %s", cacheFd)
			err = yaml.Unmarshal(cacheFd, &debConf)
			if err != nil {
				logger.Fatalf("unmarshal error: %s", err)
				return
			}
			logger.Debugf("loaded %+v", debConf)

		}

		logger.Debug("load cache.yaml", transInfo.CachePath)

		if ret, msg := CheckFileExits(transInfo.CachePath); ret {
			// load cache.yaml
			logger.Debugf("load cache: %s", transInfo.CachePath)
			cacheFd, err := ioutil.ReadFile(transInfo.CachePath)
			if err != nil {
				logger.Warnf("read error: %s %s", err, msg)
				return
			}

			err = yaml.Unmarshal(cacheFd, &configInfo)
			if err != nil {
				logger.Warnf("unmarshal error: %s", err)
				return
			}
			logger.Debugf("load config info %v", configInfo)
		} else {
			logger.Fatalf("can not found: %s", msg)
			return
		}

		// fmt.Printf("Inside rootCmd PreRun with args: %v\n", args)
		logger.Debug("mount all", configInfo.MountsItem)
		configInfo.MountsItem.DoMountALL()

		logger.Debug("configinfo.rootfsdir", configInfo.Rootfsdir)

		configInfo.debPath = transInfo.debPath
		configInfo.yamlconfig = transInfo.yamlconfig
		configInfo.Verbose = transInfo.Verbose
		configInfo.CachePath = transInfo.CachePath
		// configInfo.Workdir = transInfo.Workdir
		// configInfo.Rootfsdir = transInfo.Rootfsdir

		logger.Debug("configInfo:", configInfo)
		logger.Debug("transInfo:", transInfo)
		logger.Debug("debConf:", debConf)

		if transInfo.DebWorkdir != "" && configInfo.DebWorkdir == "" {
			configInfo.DebWorkdir = transInfo.DebWorkdir
		}

		if configInfo.DebWorkdir == "" {
			configInfo.DebWorkdir = configInfo.Workdir + "/debdir"
			if ret, _ := CheckFileExits(configInfo.DebWorkdir); !ret {
				CreateDir(configInfo.DebWorkdir)
			}
		}

		if ret, err := rfs.MountIso(configInfo.IsoPath, configInfo.Workdir+"/iso/mount"); !ret {
			logger.Error("mount iso failed!", err)
		}

		if ret, err := rfs.MountSquashfs(configInfo.Workdir+"/iso/live", configInfo.Workdir+"/iso/mount/live/filesystem.squashfs"); !ret {
			logger.Error("mount iso failed!", err)
		}

		// mount overlay to base dir
		logger.Debug("Rootfsdir:", configInfo.Rootfsdir, "runtimeBasedir:", configInfo.RuntimeBasedir, "basedir:", configInfo.Basedir, "workdir:", configInfo.Workdir)

		CreateDir(configInfo.Workdir + "/tmpdir")

		if ret, err := rfs.MountRfsWithOverlayfs(configInfo.Basedir, configInfo.Rootfsdir, configInfo.RuntimeBasedir, configInfo.Workdir+"/tmpdir", configInfo.Workdir+"/iso/live"); !ret {
			logger.Error("mount iso failed!", err)
		}
		//  umount configInfo.Rootfsdir

	},
	Run: func(cmd *cobra.Command, args []string) {
		// fmt.Printf("Inside rootCmd Run with args: %v\n", args)

		// check enter deb file
		logger.Debug("check debPath:", configInfo.debPath)
		if ret, msg := CheckFileExits(configInfo.debPath); !ret {
			logger.Fatal("can not found: ", msg)
		}

		// fetch deb file
		// DebConfig
		logger.Debugf("debConfig deb:%v", debConf.FileElement.Deb)
		for idx, _ := range debConf.FileElement.Deb {
			// fetch deb file
			debConf.FileElement.Deb[idx].FetchDebFile(configInfo.DebWorkdir)
			logger.Debugf("fetch deb path:[%d] %s", idx, debConf.FileElement.Deb[idx].Path)
			// check deb hash
			debConf.FileElement.Deb[idx].CheckDebHash()
		}

		// render DebConfig to template save to pica.sh
		logger.Debugf("render berfore %s:", debConf.FileElement.Deb)
		RenderDebConfig(debConf, configInfo.DebWorkdir+"/pica.sh")

		// chroot
		if ret, msg, err := ChrootExecShell(configInfo.Rootfsdir, configInfo.DebWorkdir+"/pica.sh", configInfo.DebWorkdir); !ret {
			logger.Fatal("chroot exec shell failed:", msg, err)
			return
		}

		var binReactor BinFormatReactor

		// fixme(heysion) set files directory
		binReactor.SearchPath = configInfo.Basedir

		// copy deb data
		// fixme(heysion): todo

		// find all elf file with path
		// FilerList := ("libc.so","lib.so")

		// get elf binary  need exclude with self path

		binReactor.GetElfList(binReactor.SearchPath + "/lib")

		excludeSoList := []string{"ld-linux-aarch64.so",
			"ld-linux-armhf.so",
			"ld-linux-x86-64.so",
			"ld-linux.so",
			"ld64.so",
			"libc.so",
			"libdl.so",
			"libgcc_s.so",
			"libm.so",
			"libstdc++.so"}
		logger.Debugf("exclude so list:", excludeSoList)

		//binReactor.FixElfLDDPath(binReactor.SearchPath + "bin/lib")
		//
		// GetFindElfMissDepends(configInfo.Basedir + "/lib")
		elfLDDLog := configInfo.DebWorkdir + "/elfldd.log"
		elfLDDShell := configInfo.DebWorkdir + "/elfldd.sh"

		logger.Debugf("out: %s , sh: %s", elfLDDLog, elfLDDShell)

		binReactor.RenderElfWithLDD(elfLDDLog, elfLDDShell)

		// // mount shell to chroot
		// logger.Debug("set output in chroot: elfldd.log")
		// if _, msg, err := ExecAndWait(10, "mount", "-B", configInfo.DebWorkdir+"/elfldd.log", configInfo.Rootfsdir+configInfo.DebWorkdir+"/elfldd.log"); err != nil {
		// 	logger.Fatalf("mount %s to %s failed! ", configInfo.Rootfsdir+configInfo.DebWorkdir+"/elfldd.log", err, msg)
		// }

		// chroot
		if ret, msg, err := ChrootExecShell(configInfo.Rootfsdir, elfLDDShell, configInfo.DebWorkdir); !ret {
			logger.Fatal("chroot exec shell failed:", msg, err)
			return
		}

		// read elfldd.log
		logger.Debug("read elfldd.log")
		if elfLDDLogFile, err := os.Open(elfLDDLog); err != nil {
			logger.Fatal("open elfldd.log failed:", err)
			//elfLDDLogFile.Close()
		} else {
			defer elfLDDLogFile.Close()

			LogFileItor := bufio.NewScanner(elfLDDLogFile)
			LogFileItor.Split(bufio.ScanLines)
			var ReadLine string
			for LogFileItor.Scan() {
				ReadLine = LogFileItor.Text()
				if len(ReadLine) > 0 && func() bool {
					for _, v := range excludeSoList {
						if strings.HasSuffix(ReadLine, v) {
							return false
						}
					}
					return true
				}() {
					logger.Debugf("%s", ReadLine)
					binReactor.ElfNeedPath[ReadLine] = 1
				}
			}

		}

		// fix library
		// if msg, ret := GetElfNeedWithLDD("/bin/bash"); ret != nil {
		// 	logger.Debug("get elf need failed: ", msg)
		// }

		// fix desktop
		// FixDesktop()

		// update info.json

		// umount

		// build uab
		// ll-builder export --local

	},
	PostRun: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Inside rootCmd PostRun with args: %v\n", args)
		//configInfo.MountsItem.DoUmountAOnce()

		// umount overlayfs
		logger.Debug("umount rootfs")
		if ret, err := rfs.UmountRfs(configInfo.Rootfsdir); !ret {
			logger.Error("mount iso failed!", err)
		}

		// umount squashfs
		logger.Debug("umount squashfs")
		if ret, err := rfs.UmountSquashfs(configInfo.Workdir + "/iso/live"); !ret {
			logger.Error("mount iso failed!", err)
		}

		// umount iso
		logger.Debug("umount iso")
		if ret, err := rfs.UmountIso(configInfo.Workdir + "/iso/mount"); !ret {
			logger.Error("mount iso failed!", err)
		}
		//  umount configInfo.Rootfsdir
		logger.Debug("umount mounts devs")
		configInfo.MountsItem.DoUmountAOnce()

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

	initCmd.Flags().StringVarP(&configInfo.Config, "config", "c", "", "config")
	initCmd.Flags().StringVarP(&configInfo.Workdir, "workdir", "w", "", "work directory")
	//initCmd.Flags().StringVarP(&configInfo.iso, "iso", "f", "", "iso")
	initCmd.Flags().BoolVarP(&configInfo.Cache, "keep-cached", "k", true, "keep cached")
	err := initCmd.MarkFlagRequired("config")
	if err != nil {
		logger.Fatal("config required failed", err)
		return
	}

	// convert cmd add
	rootCmd.AddCommand(convertCmd)
	convertCmd.Flags().StringVarP(&transInfo.yamlconfig, "config", "c", "", "config")
	convertCmd.Flags().StringVarP(&transInfo.Workdir, "workdir", "w", "", "work directory")
	convertCmd.Flags().StringVarP(&transInfo.CachePath, "cache-file", "f", "", "cache yaml file")
	convertCmd.Flags().StringVarP(&transInfo.debPath, "deb-file", "d", "", "deb file")

	err = convertCmd.MarkFlagRequired("config")
	if err != nil {
		logger.Fatal("yaml config required failed", err)
	}
	err = convertCmd.MarkFlagRequired("deb-file")
	if err != nil {
		logger.Fatal("deb file required failed", err)
	}

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// root cmd add
	rootCmd.PersistentFlags().BoolVarP(&configInfo.Verbose, "verbose", "v", false, "verbose output")

	if configInfo.Workdir == "" {
		configInfo.Workdir = "/mnt/workdir"
	}

	if transInfo.Workdir == "" {
		transInfo.Workdir = "/mnt/workdir"
	}
	// fix cache path
	if transInfo.CachePath == "" {
		transInfo.CachePath = transInfo.Workdir + "/cache.yaml"
	}

	configInfo.MountsItem.Mounts = make(map[string]MountItem)

	err = rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
