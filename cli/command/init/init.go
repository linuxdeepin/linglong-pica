/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package init

import (
	"github.com/spf13/cobra"
	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/cli/config"
	"pkg.deepin.com/linglong/pica/cli/deb"
	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

type initOptions struct {
	comm.Options
	getType     string
	packageId   string
	packageName string
	comm.Config
}

func NewInitCommand() *cobra.Command {
	var options initOptions
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a deb conversion template",
		Long: `Generate package.yaml for ll-pica deb conversion.

The generated template only keeps the fields that are still used by the
current deb workflow. runtime.runtime_version is empty by default and is only
needed for Qt/Dtk-based projects.`,
		Example: `  ll-pica deb init -w work
  ll-pica deb init -w work --pi com.example.app --pn com.example.app -t repo
  ll-pica deb init -w work --pi com.example.app --pn example -t local`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(&options)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.Options.Config, "config", "c", "", "package.yaml path; defaults to <workdir>/package.yaml")
	flags.StringVarP(&options.Workdir, "workdir", "w", "", "working directory used for package.yaml and conversion outputs")
	flags.StringVar(&options.RuntimeVersion, "rv", "", "runtime.runtime_version; leave empty unless the app needs a Qt/Dtk runtime")
	flags.StringVar(&options.BaseVersion, "bv", "", "base version written to package.yaml")
	flags.StringVarP(&options.Source, "source", "s", "", "deprecated runtime source field; kept only for reading legacy config")
	flags.StringVar(&options.DistroVersion, "dv", "", "deprecated distro field; kept only for reading legacy config")
	flags.StringVarP(&options.Arch, "arch", "a", "", "target architecture recorded in package.yaml, for example amd64 or arm64")
	flags.StringVarP(&options.getType, "type", "t", "", "deb source type written into the template: repo or local")
	flags.StringVar(&options.packageId, "pi", "", "Linglong package id, for example com.example.app")
	flags.StringVar(&options.packageName, "pn", "", "deb package name used by apt or the local package metadata")
	return cmd
}

func runInit(options *initOptions) error {
	options.Workdir = comm.WorkPath(options.Workdir)
	configFilePath := comm.ConfigFilePath(options.Workdir, options.Options.Config)

	// 创建工作目录
	comm.InitWorkDir(options.Workdir)
	// 创建 ~/.pica 目录
	comm.InitPicaConfigDir()

	packConf := config.NewPackConfig()

	// 如果不存在 pica 配置文件，生成一份默认配置
	if ret, _ := fs.CheckFileExits(comm.PicaConfigJsonPath()); !ret {
		log.Logger.Errorf("%s can not found", comm.PicaConfigJsonPath())
		packConf.Runtime.SaveOrUpdateConfigJson(comm.PicaConfigJsonPath())
	} else {
		// 如果存在 pica 配置文件解析配置文件
		packConf.Runtime.ReadConfigJson()
	}
	// runtime 默认留空，只有显式指定时才写入 package.yaml
	packConf.Runtime.Config.RuntimeVersion = ""

	assign := func(config *string, option string) {
		if option != "" {
			*config = option
		}
	}
	if options.BaseVersion != "" || options.RuntimeVersion != "" || options.Source != "" || options.DistroVersion != "" || options.Arch != "" {
		assign(&packConf.Runtime.Config.RuntimeVersion, options.RuntimeVersion)
		assign(&packConf.Runtime.Config.Source, options.Source)
		assign(&packConf.Runtime.Config.DistroVersion, options.DistroVersion)
		assign(&packConf.Runtime.Config.Arch, options.Arch)
		assign(&packConf.Runtime.Config.BaseVersion, options.BaseVersion)
		packConf.Runtime.Config.SaveOrUpdateConfigJson(comm.PicaConfigJsonPath())
	}

	if options.packageId != "" && options.packageName != "" && options.getType != "" {
		if err := comm.ValidatePackageID(options.packageId); err != nil {
			return err
		}
		packConf.File.Deb = []deb.Deb{
			{
				Type: options.getType,
				Id:   options.packageId,
				Name: options.packageName,
			},
		}
	}

	packConf.CreatePackConfigYaml(configFilePath)
	return nil
}
