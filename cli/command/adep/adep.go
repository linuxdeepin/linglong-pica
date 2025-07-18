/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package adep

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/cli/config"
	"pkg.deepin.com/linglong/pica/cli/linglong"
	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

type adepOptions struct {
	path    string
	deps    string
	withDep bool // 带上依赖树
}

func NewADepCommand() *cobra.Command {
	var options adepOptions
	cmd := &cobra.Command{
		Use:   "adep",
		Short: "Add dependency packages to linglong.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdep(&options)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.deps, "deps", "d", "", "dependencies to be added, separator is ','")
	flags.StringVarP(&options.path, "path", "p", "linglong.yaml", "path to linglong.yaml")
	flags.BoolVar(&options.withDep, "withDep", false, "Add dependency tree")
	return cmd
}

func runAdep(options *adepOptions) error {
	if options.deps == "" {
		log.Logger.Fatal("The parameter d has not been set ")
	}

	path, err := filepath.Abs(options.path)
	if ret, _ := fs.CheckFileExits(path); !ret {
		log.Logger.Errorf("%s not found", path)
		return err
	}

	packConfig := config.NewPackConfig()
	// 如果不存在 pica 配置文件，生成一份默认配置
	if ret, _ := fs.CheckFileExits(comm.PicaConfigJsonPath()); !ret {
		log.Logger.Infof("%s can not found", comm.PicaConfigJsonPath())
		packConfig.Runtime.SaveOrUpdateConfigJson(comm.PicaConfigJsonPath())
	} else {
		// 如果存在 pica 配置文件解析配置文件
		packConfig.Runtime.ReadConfigJson()
	}

	builder := linglong.NewLinglongBuilder()
	builder.ReadLinglongYaml(path)
	// 读入的 build 字段是一整个字符串，需要用换行符切割成数组，并且读入的字符串最后一行会有换行符号，需要去掉
	builder.Build = strings.Split(strings.TrimSuffix(builder.BuildInput, "\n"), "\n")
	// 读入的 Description 中包含换行符，需要替换掉
	builder.Package.Description = strings.TrimSuffix(builder.Package.Description, "\n")

	depList := strings.Split(options.deps, ",")
	allDepends := append(builder.BuildExt.Apt.Depends, depList...)
	builder.BuildExt.Apt.Depends = comm.RemoveExcessDepends(allDepends)
	if builder.CreateLinglongYaml(path) {
		log.Logger.Infof("generate %s success.", comm.LinglongYaml)
	} else {
		log.Logger.Errorf("generate %s failed", comm.LinglongYaml)
	}
	return nil
}
