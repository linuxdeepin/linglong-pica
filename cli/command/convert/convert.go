/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package convert

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"pault.ag/go/debian/control"
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
	exportFile  string
}

func NewConvertCommand() *cobra.Command {
	var options convertOptions
	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert deb packages into Linglong build inputs",
		Long: `Generate linglong.yaml and unpacked sources for a deb package.

You can convert from package.yaml, from a repo package name, or from a local
.deb file path passed with -c. Dependencies are written into buildext.apt
instead of sources. Architecture mismatch is checked only before ll-builder
build, so convert-only workflows remain available for cross-build scenarios.`,
		Example: `  ll-pica deb convert -w work
  ll-pica deb convert -w work -b
  ll-pica deb convert -c ./com.example.app_1.0.0_amd64.deb -w work
  ll-pica deb convert -w work --pi com.example.app --pn com.example.app`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConvert(&options)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.Config, "config", "c", "", "package.yaml path, or a local .deb file path")
	flags.StringVarP(&options.Workdir, "workdir", "w", "", "working directory used for downloads, unpacked files, and linglong.yaml")
	flags.StringVarP(&options.gtype, "type", "t", "local", "deb source type when creating config on the fly: local or repo")
	flags.StringVar(&options.packageId, "pi", "", "Linglong package id used for inline repo conversion")
	flags.StringVar(&options.packageName, "pn", "", "deb package name used for inline repo conversion")
	flags.BoolVarP(&options.buildFlag, "build", "b", false, "run ll-builder build and export after generating linglong.yaml")
	flags.StringVar(&options.exportFile, "exportFile", "uab", "export format after --build: uab or layer")
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
		packConfig.Runtime.Config.RuntimeVersion = ""
		ret, err := deb.AptShow(configFilePath)
		if err == nil {
			info, err := control.ParseControl(bufio.NewReader(strings.NewReader(ret)), "")
			if err != nil {
				log.Logger.Warnf("parse control error: %s", err)
				return err
			}

			packConfig.File.Deb = []deb.Deb{
				{
					Type: options.gtype,
					Id:   info.Source.Paragraph.Values["Package"],
					Ref:  configFilePath,
					Name: info.Source.Paragraph.Values["Package"],
				},
			}
			// 此时替换 configFilePath 为 工作目录的 package.yaml
			configFilePath = comm.ConfigFilePath(options.Workdir, "")
			packConfig.CreatePackConfigYaml(configFilePath)
		}
	}

	if options.packageId != "" && options.packageName != "" {
		packConfig.Runtime.Config.RuntimeVersion = ""
		if err := comm.ValidatePackageID(options.packageId); err != nil {
			return err
		}
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
		if err := comm.ValidatePackageID(packConfig.File.Deb[idx].Id); err != nil {
			return err
		}
		log.Logger.Infof("starting conversion for package %s", packConfig.File.Deb[idx].Id)
		appPath := filepath.Join(comm.BuildPackPath(options.Workdir), packConfig.File.Deb[idx].Id)
		linglongYamlPath := filepath.Join(appPath, comm.LinglongYaml)

		// 如果已经存在 linglong.yaml 文件直接跳过。
		if ret, err := fs.CheckFileExits(linglongYamlPath); ret && err == nil {
			log.Logger.Infof("%s: linglong.yaml already exists, skip generation", packConfig.File.Deb[idx].Id)
			continue
		}

		fs.CreateDir(appPath)
		// repo 类型通过 apt download 查询主包 URL，本地 deb 直接复制到工作目录。
		if packConfig.File.Deb[idx].Ref == "" {
			packConfig.File.Deb[idx].Ref = packConfig.File.Deb[idx].GetPackageUrl(packConfig.Runtime.Source, packConfig.Runtime.DistroVersion, packConfig.Runtime.Arch)
			if packConfig.File.Deb[idx].Ref == "" {
				return fmt.Errorf("%s: get package url failed", packConfig.File.Deb[idx].Name)
			}
			packConfig.File.Deb[idx].Path = filepath.Join(comm.LocalPackageSourceDir(appPath), filepath.Base(packConfig.File.Deb[idx].Ref))
		}
		// fetch deb file
		if len(packConfig.File.Deb[idx].Ref) > 0 {
			packConfig.File.Deb[idx].Path = filepath.Join(comm.LocalPackageSourceDir(appPath), filepath.Base(packConfig.File.Deb[idx].Ref))

			if ret, _ := fs.CheckFileExits(packConfig.File.Deb[idx].Path); ret {
				if hash := packConfig.File.Deb[idx].CheckDebHash(); hash {
					log.Logger.Infof("download skipped because of %s cached", packConfig.File.Deb[idx].Name)
				} else {
					log.Logger.Warnf("check deb hash failed! : ", packConfig.File.Deb[idx].Name)
					fs.RemovePath(packConfig.File.Deb[idx].Path)

					if ok := packConfig.File.Deb[idx].FetchDebFile(packConfig.File.Deb[idx].Path); !ok {
						return fmt.Errorf("%s: fetch deb file failed", packConfig.File.Deb[idx].Name)
					}
					log.Logger.Debugf("fetch deb path:[%d] %s", idx, packConfig.File.Deb[idx].Path)

					if ret := packConfig.File.Deb[idx].CheckDebHash(); !ret {
						return fmt.Errorf("%s: check deb hash failed", packConfig.File.Deb[idx].Name)
					}
					log.Logger.Infof("download %s success.", packConfig.File.Deb[idx].Name)
				}
			} else {
				if ok := packConfig.File.Deb[idx].FetchDebFile(packConfig.File.Deb[idx].Path); !ok {
					return fmt.Errorf("%s: fetch deb file failed", packConfig.File.Deb[idx].Name)
				}
				log.Logger.Infof("fetch deb path:[%d] %s", idx, packConfig.File.Deb[idx].Path)

				if ret := packConfig.File.Deb[idx].CheckDebHash(); !ret {
					return fmt.Errorf("%s: check deb hash failed", packConfig.File.Deb[idx].Name)
				}
				log.Logger.Infof("download %s success.", packConfig.File.Deb[idx].Name)
			}

			// 提取 deb 包的相关数据
			if err := packConfig.File.Deb[idx].ExtractDeb(); err != nil {
				return err
			}

			// 依赖处理
			packConfig.File.Deb[idx].ResolveDepends()
			// 生成构建脚本
			packConfig.File.Deb[idx].GenerateBuildScript()
			// 对 linglong.yaml 依赖去重
			packConfig.File.Deb[idx].RuntimeDepends = comm.RemoveExcessDepends(packConfig.File.Deb[idx].RuntimeDepends)
			packConfig.File.Deb[idx].BuildDepends = comm.RemoveExcessDepends(packConfig.File.Deb[idx].BuildDepends)

			builder := linglong.LinglongBuilder{
				Package: linglong.Package{
					Appid:       packConfig.File.Deb[idx].Id,
					Name:        packConfig.File.Deb[idx].Name,
					Version:     packConfig.File.Deb[idx].Version,
					Kind:        packConfig.File.Deb[idx].PackageKind,
					Description: packConfig.File.Deb[idx].Desc,
				},
				Runtime: formatRuntimeRef(packConfig.Runtime.Id, packConfig.Runtime.RuntimeVersion),
				Base:    fmt.Sprintf("%s/%s", packConfig.Runtime.BaseId, packConfig.Runtime.BaseVersion),
				Command: packConfig.File.Deb[idx].Command,
				Build:   packConfig.File.Deb[idx].Build,
				BuildExt: linglong.BuildExt{
					Apt: linglong.AptExt{
						BuildDepends: packConfig.File.Deb[idx].BuildDepends,
						Depends:      packConfig.File.Deb[idx].RuntimeDepends,
					},
				},
			}

			log.Logger.Infof("%s: generating linglong.yaml", packConfig.File.Deb[idx].Id)
			// 生成 linglong.yaml 文件
			if builder.CreateLinglongYaml(linglongYamlPath) {
				log.Logger.Infof("%s: generated linglong.yaml", packConfig.File.Deb[idx].Id)
			} else {
				log.Logger.Errorf("generate %s failed", comm.LinglongYaml)
			}

			// 构建玲珑包
			if options.buildFlag {
				if err := validateDebArchitecture(packConfig.File.Deb[idx].Architecture, packConfig.Runtime.Arch); err != nil {
					return fmt.Errorf("%s: %w", packConfig.File.Deb[idx].Name, err)
				}
				buildLinglongPath := filepath.Dir(linglongYamlPath)
				log.Logger.Infof("%s: building package", packConfig.File.Deb[idx].Id)
				builder.LinglongBuild(buildLinglongPath, "ll-builder build")
				log.Logger.Infof("%s: exporting package (%s)", packConfig.File.Deb[idx].Id, options.exportFile)
				builder.LinglongExport(buildLinglongPath, options.exportFile)
				log.Logger.Infof("%s: export completed", packConfig.File.Deb[idx].Id)
			} else {
				log.Logger.Infof("%s: skip build/export (set --build to enable)", packConfig.File.Deb[idx].Id)
			}
		}
	}
	return nil
}

func formatRuntimeRef(id, version string) string {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(version) == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", id, version)
}

func validateDebArchitecture(debArch, targetArch string) error {
	debArch = strings.TrimSpace(debArch)
	targetArch = strings.TrimSpace(targetArch)
	if debArch == "" || targetArch == "" {
		return nil
	}
	if debArch == targetArch {
		return nil
	}
	return fmt.Errorf("deb architecture %q does not match target architecture %q", debArch, targetArch)
}
