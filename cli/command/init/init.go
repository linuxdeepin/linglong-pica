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
	config.Config
}

func NewInitCommand() *cobra.Command {
	var options initOptions
	cmd := &cobra.Command{
		Use:   "init",
		Short: "init config template",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(&options)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.Options.Config, "config", "c", "", "config file")
	flags.StringVarP(&options.Workdir, "workdir", "w", "", "work directory")
	flags.StringVar(&options.Version, "rv", "", "runtime version")
	flags.StringVar(&options.BaseVersion, "bv", "", "base version")
	flags.StringVarP(&options.Source, "source", "s", "", "runtime source")
	flags.StringVar(&options.DistroVersion, "dv", "", "distribution Version")
	flags.StringVarP(&options.Arch, "arch", "a", "", "runtime arch")
	flags.StringVarP(&options.getType, "type", "t", "", "get type")
	flags.StringVar(&options.packageId, "pi", "", "package id")
	flags.StringVar(&options.packageName, "pn", "", "package name")
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

	assign := func(config *string, option string) {
		if option != "" {
			*config = option
		}
	}
	if options.BaseVersion != "" || options.Version != "" || options.Source != "" || options.DistroVersion != "" || options.Arch != "" {
		assign(&packConf.Runtime.Config.Version, options.Version)
		assign(&packConf.Runtime.Config.Source, options.Source)
		assign(&packConf.Runtime.Config.DistroVersion, options.DistroVersion)
		assign(&packConf.Runtime.Config.Arch, options.Arch)
		assign(&packConf.Runtime.Config.BaseVersion, options.BaseVersion)
		packConf.Runtime.Config.SaveOrUpdateConfigJson(comm.PicaConfigJsonPath())
	}

	if options.packageId != "" && options.packageName != "" && options.getType != "" {
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
