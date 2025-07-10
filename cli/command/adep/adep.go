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
	"pkg.deepin.com/linglong/pica/cli/deb"
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

// generateDepProcessScript 生成依赖包处理脚本（不包含主包处理逻辑）
func generateDepProcessScript() []string {
	return []string{
		"",
		"# Process dependency packages added by ll-pica adep",
		"SOURCES=\"/project/linglong/sources\"",
		"",
		"# Extract and process dependency deb packages",
		"OUT_DIR=\"$(mktemp -d)\"",
		"DEPS_LIST=\"$OUT_DIR/DEPS.list\"",
		"find $SOURCES -type f -name \"*.deb\" > $DEPS_LIST",
		"DATA_LIST_DIR=\"$OUT_DIR/data\"",
		"mkdir -p /tmp/deb-source-file",
		"",
		"",
		"while IFS= read -r file",
		"do",
		"    echo \"Processing dependency: $file\"",
		"    CONTROL_FILE=$(ar -t $file | grep control.tar)",
		"    ar -x \"$file\" $CONTROL_FILE",
		"    PKG=$(tar -xf $CONTROL_FILE ./control -O | grep '^Package:' | awk '{print $2}')",
		"    rm $CONTROL_FILE || true",
		"    DATA_FILE=$(ar -t $file | grep data.tar)",
		"    ar -x $file $DATA_FILE",
		"    mkdir -p $DATA_LIST_DIR",
		"    tar -xvf $DATA_FILE -C $DATA_LIST_DIR >> \"/tmp/deb-source-file/$(basename $file).list\"",
		"    rm -rf $DATA_FILE 2>/dev/null || true",
		"    rm -r ${DATA_LIST_DIR:?}/usr/share/applications* 2>/dev/null || true",
		"    sed -i \"s#/usr#$PREFIX#g\" $DATA_LIST_DIR/usr/lib/$TRIPLET/pkgconfig/*.pc 2>/dev/null || true",
		"    sed -i \"s#/usr#$PREFIX#g\" $DATA_LIST_DIR/usr/share/pkgconfig/*.pc 2>/dev/null || true",
		"    find $DATA_LIST_DIR -type l | while IFS= read -r file; do",
		"        Link_Target=$(readlink $file)",
		"        if echo $Link_Target | grep -q ^/lib && ! [ -f $Link_Target ]; then",
		"            ln -sf $PREFIX$Link_Target $file",
		"            echo \"    FIX LINK $Link_Target => $PREFIX$Link_Target\"",
		"        fi",
		"    done",
		"    find $DATA_LIST_DIR -type f -exec file {} \\; | grep 'shared object' | awk -F: '{print $1}' | while IFS= read -r file; do",
		"        runpath=$(readelf -d $file | grep RUNPATH |  awk '{print $NF}')",
		"        if echo $runpath | grep -q '^\\[/'; then",
		"            runpath=${runpath#[}",
		"            runpath=${runpath%]}",
		"            newRunpath=${runpath//usr\\/lib/runtime\\/lib}",
		"            newRunpath=${newRunpath//usr/runtime}",
		"            patchelf --set-rpath $newRunpath $file",
		"            echo \"    FIX RUNPATH $file $runpath => $newRunpath\"",
		"        fi",
		"    done",
		"    # 只复制usr目录下的文件",
		"    cp -rP $DATA_LIST_DIR/usr/* $PREFIX/ 2>/dev/null || true",
		"done < \"$DEPS_LIST\"",
		"rm -r $OUT_DIR || true",
	}
}

// hasDepProcessing 检查build脚本中是否已经包含依赖处理逻辑
func hasDepProcessing(buildLines []string) bool {
	for _, line := range buildLines {
		if strings.Contains(line, "Process dependency packages") {
			return true
		}
	}
	return false
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

	// 检查是否已经有依赖处理脚本
	alreadyHasDepProcessing := hasDepProcessing(builder.Build)

	depList := strings.Split(options.deps, ",")
	var allNewSources []comm.Source

	for _, dep := range depList {
		deb := deb.Deb{
			Name:         dep,
			Depends:      dep,
			Architecture: packConfig.Runtime.Arch,
			Path:         filepath.Join(comm.LLSourcePath(filepath.Dir(options.path)), "app"),
		}
		// 添加包本身
		deb.GetPackageUrl(packConfig.Runtime.Source, packConfig.Runtime.DistroVersion, packConfig.Runtime.Arch)
		deb.ResolveDepends(packConfig.Runtime.Source, packConfig.Runtime.DistroVersion, options.withDep)
		allNewSources = append(allNewSources, deb.Sources...)
	}

	// 如果有新的依赖包且还没有处理脚本，则添加依赖处理脚本
	if len(allNewSources) > 0 && !alreadyHasDepProcessing {
		// 生成依赖处理脚本
		depProcessScript := generateDepProcessScript()
		builder.Build = append(builder.Build, depProcessScript...)
	}

	// 添加所有新的sources
	builder.Sources = append(builder.Sources, allNewSources...)

	// 对 linglong.yaml 依赖去重
	builder.Sources = comm.RemoveExcessDeps(builder.Sources)

	if builder.CreateLinglongYaml(path) {
		log.Logger.Infof("generate %s success.", comm.LinglongYaml)
	} else {
		log.Logger.Errorf("generate %s failed", comm.LinglongYaml)
	}
	return nil
}
