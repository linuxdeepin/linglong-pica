/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package convert

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/cli/config"
	"pkg.deepin.com/linglong/pica/cli/deb"
	"pkg.deepin.com/linglong/pica/cli/linglong"
	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

type convertOptions struct {
	comm.Options
	gtype       string
	packageId   string
	packageName string
	buildFlag   bool
}

func NewConvertCommand() *cobra.Command {
	var options convertOptions
	cmd := &cobra.Command{
		Use:          "convert",
		Short:        "Convert deb to uab",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConvert(&options)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.Config, "config", "c", "", "config file")
	flags.StringVarP(&options.Workdir, "workdir", "w", "", "work directory")
	flags.StringVarP(&options.gtype, "type", "t", "local", "get app type")
	flags.StringVar(&options.packageId, "pi", "", "package id")
	flags.StringVar(&options.packageName, "pn", "", "package name")
	flags.BoolVarP(&options.buildFlag, "build", "b", false, "build linglong")
	return cmd
}

func runConvert(options *convertOptions) error {
	options.Workdir = comm.WorkPath(options.Workdir)
	configFilePath := comm.ConfigFilePath(options.Workdir, options.Config)

	comm.InitWorkDir(options.Workdir)
	comm.InitPicaConfigDir()

	packConfig := config.NewPackConfig()
	// 如果不存在 pica 配置文件，生成一份默认配置
	if ret, _ := fs.CheckFileExits(comm.PicaConfigJsonPath()); !ret {
		log.Logger.Infof("%s can not found", comm.PicaConfigJsonPath())
		packConfig.Runtime.SaveOrUpdateConfigJson(comm.PicaConfigJsonPath())
	} else {
		// 如果存在 pica 配置文件解析配置文件
		packConfig.Runtime.ReadConfigJson()
	}

	// 如果传入的是 deb 包， 先构造一下 package.yaml 文件
	if strings.HasSuffix(options.Config, ".deb") {
		ret, err := deb.AptShow(configFilePath)
		if err == nil {
			var d deb.Deb

			// apt-cache show Unmarshal
			err = yaml.Unmarshal([]byte(ret), &d)
			if err != nil {
				log.Logger.Warnf("apt-cache show unmarshal error: %s", err)
			}
			packConfig.File.Deb = []deb.Deb{
				{
					Type: options.gtype,
					Id:   d.Package,
					Ref:  configFilePath,
					Name: d.Package,
				},
			}
			// 此时替换 configFilePath 为 工作目录的 package.yaml
			configFilePath = comm.ConfigFilePath(options.Workdir, "")
			packConfig.CreatePackConfigYaml(configFilePath)
		}
	}

	if options.packageId != "" && options.packageName != "" {
		packConfig.File.Deb = []deb.Deb{
			{
				Type: "repo",
				Id:   options.packageId,
				Name: options.packageName,
			},
		}
		packConfig.CreatePackConfigYaml(configFilePath)
	}

	if ret := packConfig.ReadPackConfigYaml(configFilePath); !ret {
		log.Logger.Fatalf("read pack config yaml error")
	}

	for idx := range packConfig.File.Deb {
		appPath := filepath.Join(comm.BuildPackPath(options.Workdir), packConfig.File.Deb[idx].Id, packConfig.Runtime.Arch)
		linglongYamlPath := filepath.Join(appPath, comm.LinglongYaml)

		// 如果已经存在 linglong.yaml 文件直接跳过。
		if ret, err := fs.CheckFileExits(linglongYamlPath); ret && err == nil {
			log.Logger.Infof("%s file already exists", linglongYamlPath)
			continue
		}

		fs.CreateDir(appPath)
		// 如果 Ref 为空，type 为 repo, 那么先使用 aptly 获取 url 链接， 如果没有就使用 apt download 获取 url 链接，
		// 另外的如果 type 为 local 直接将 deb 包下载到工作目录
		if packConfig.File.Deb[idx].Ref == "" {
			packConfig.File.Deb[idx].Ref = packConfig.File.Deb[idx].GetPackageUrl(packConfig.Runtime.Source, packConfig.Runtime.DistroVersion, packConfig.Runtime.Arch)
			if packConfig.File.Deb[idx].Ref == "" {
				log.Logger.Fatalf("get package url failed")
			}
			packConfig.File.Deb[idx].Path = filepath.Join(comm.LLSourcePath(appPath), filepath.Base(packConfig.File.Deb[idx].Ref))
		}
		// fetch deb file
		if len(packConfig.File.Deb[idx].Ref) > 0 {
			packConfig.File.Deb[idx].Path = filepath.Join(comm.LLSourcePath(appPath), filepath.Base(packConfig.File.Deb[idx].Ref))

			if ret, _ := fs.CheckFileExits(packConfig.File.Deb[idx].Path); ret {
				if hash := packConfig.File.Deb[idx].CheckDebHash(); hash {
					log.Logger.Infof("download skipped because of %s cached", packConfig.File.Deb[idx].Name)
				} else {
					log.Logger.Warnf("check deb hash failed! : ", packConfig.File.Deb[idx].Name)
					fs.RemovePath(packConfig.File.Deb[idx].Path)

					packConfig.File.Deb[idx].FetchDebFile(packConfig.File.Deb[idx].Path)
					log.Logger.Debugf("fetch deb path:[%d] %s", idx, packConfig.File.Deb[idx].Path)

					if ret := packConfig.File.Deb[idx].CheckDebHash(); !ret {
						log.Logger.Warnf("check deb hash failed! : ", packConfig.File.Deb[idx].Name)
						continue
					}
					log.Logger.Infof("download %s success.", packConfig.File.Deb[idx].Name)
				}
			} else {
				packConfig.File.Deb[idx].FetchDebFile(packConfig.File.Deb[idx].Path)
				log.Logger.Infof("fetch deb path:[%d] %s", idx, packConfig.File.Deb[idx].Path)

				if ret := packConfig.File.Deb[idx].CheckDebHash(); !ret {
					log.Logger.Warnf("check deb hash failed! : ", packConfig.File.Deb[idx].Name)
					continue
				}
				log.Logger.Infof("download %s success.", packConfig.File.Deb[idx].Name)
			}

			// 提取 deb 包的相关数据
			if err := packConfig.File.Deb[idx].ExtractDeb(); err != nil {
				return err
			}

			// 可能存在依赖为空的情况
			if packConfig.File.Deb[idx].Depends != "" {
				// 依赖处理
				packConfig.File.Deb[idx].ResolveDepends(packConfig.Runtime.Source, packConfig.Runtime.DistroVersion)
			}
			// 生成构建脚本
			packConfig.File.Deb[idx].GenerateBuildScript()
			// linglong.yaml 依赖去重
			packConfig.File.Deb[idx].RemoveExcessDeps()

			builder := linglong.LinglongBuilder{
				Package: linglong.Package{
					Appid:       packConfig.File.Deb[idx].Id,
					Name:        packConfig.File.Deb[idx].Name,
					Version:     packConfig.File.Deb[idx].Version,
					Kind:        packConfig.File.Deb[idx].PackageKind,
					Description: packConfig.File.Deb[idx].Name,
				},
				Runtime: fmt.Sprintf("%s/%s", packConfig.Runtime.Id, packConfig.Runtime.Version),
				Base:    fmt.Sprintf("%s/%s", packConfig.Runtime.BaseId, packConfig.Runtime.Version),
				Command: []string{
					packConfig.File.Deb[idx].Command,
				},
				Sources: packConfig.File.Deb[idx].Sources,
				Build:   packConfig.File.Deb[idx].Build,
			}

			// 生成 linglong.yaml 文件
			if builder.CreateLinglongYaml(linglongYamlPath) {
				log.Logger.Infof("generate %s success.", comm.LinglongYaml)
			} else {
				log.Logger.Errorf("generate %s failed", comm.LinglongYaml)
			}

			// 构建玲珑包
			if options.buildFlag {
				if ret, msg, err := comm.ExecAndWait(10, "sh", "-c",
					fmt.Sprintf("cd %s && ll-builder build", appPath)); err != nil {
					log.Logger.Infof("build %s success.", packConfig.File.Deb[idx].Name)
				} else {
					log.Logger.Warnf("msg: %+v err:%+v, out: %+v", msg, err, ret)
				}

				// 导出玲珑包
				if ret, msg, err := comm.ExecAndWait(10, "sh", "-c",
					fmt.Sprintf("cd %s && ll-builder export", appPath)); err != nil {
					log.Logger.Infof("%s export success.", packConfig.File.Deb[idx].Name)
				} else {
					log.Logger.Warnf("msg: %+v err:%+v, out: %+v", msg, err, ret)
				}
			}
		}
	}
	return nil
}
