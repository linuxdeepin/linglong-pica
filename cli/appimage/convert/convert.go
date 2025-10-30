/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package convert

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/cli/linglong"
	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

type convertOptions struct {
	packageId          string
	packageName        string
	packageVersion     string
	packageDescription string
	appimageFile       string
	appimageFileUrl    string
	appimageFileHash   string
	buildFlag          bool
	exportLayerFlag    bool
}

func NewConvertCommand() *cobra.Command {
	var options convertOptions

	cmd := &cobra.Command{
		Use:          "convert",
		Short:        "Convert appimage to uab",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConvert(&options)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.packageId, "id", "i", "", "the unique name of the package")
	flags.StringVarP(&options.packageName, "name", "n", "", "the description the package")
	flags.StringVarP(&options.packageVersion, "version", "v", "", "the version of the package")
	flags.StringVarP(&options.packageDescription, "description", "d", "", "detailed description of the package")
	flags.StringVarP(&options.appimageFile, "file", "f", "", `app package file, it not required option,
you can ignore this option
when you set --url option and --hash option`)
	flags.StringVarP(&options.appimageFileUrl, "url", "u", "", "pkg url, it not required option,you can ignore this option when you set -f option")
	flags.StringVarP(&options.appimageFileHash, "hash", "", "", "pkg hash value, it must be used with --url option")
	flags.BoolVarP(&options.buildFlag, "build", "b", false, "build linglong")
	flags.BoolVarP(&options.exportLayerFlag, "layer", "l", false, "export layer file")
	return cmd
}

func runConvert(options *convertOptions) error {
	if options.packageId == "" {
		return fmt.Errorf("package id is required")
	}

	if options.packageVersion == "" {
		return fmt.Errorf("package version is required")
	}

	if options.appimageFile == "" && options.appimageFileUrl == "" {
		return fmt.Errorf("file option or url option is required")
	}

	if options.appimageFileUrl != "" && options.appimageFileHash == "" {
		return fmt.Errorf("hash option is required when use url option")
	}

	if options.packageName == "" {
		options.packageName = options.packageId
	}

	if options.packageDescription == "" {
		options.packageDescription = "converted from appimage"
	}

	var suffix string
	if options.appimageFile != "" {
		suffix = path.Ext(options.appimageFile)
	} else {
		suffix = path.Ext(options.appimageFileUrl)
	}

	appImageFileType := suffix == ".AppImage" || suffix == ".appimage"

	if !appImageFileType {
		return fmt.Errorf("appimage file must be .AppImage or .appimage")
	}

	var build []string

	build = append(build, []string{
		"cd sources",
		"APPIMAGE=$(find . -regex '.*\\.AppImage\\|.*appimage' -exec basename {} \\;)",
		"chmod +x ${APPIMAGE}",
		"./${APPIMAGE} --appimage-extract",
		"BINNAME=${LINGLONG_APPID}",
		"APP_PREFIX=${BINNAME}",
		"echo \"#!/usr/bin/env bash\" > ${BINNAME}",
		"echo \"unset LD_LIBRARY_PATH\" >> ${BINNAME}",
		"echo \"cd ${PREFIX}/lib/${APP_PREFIX} && ./AppRun $@\" >> ${BINNAME}",
		"# only search for .desktop file in the squashfs-root directory",
		"DESKTOP_FILE=$(find squashfs-root -maxdepth 1 -regex '.*\\.desktop' -exec basename {} \\;)",
		"cp squashfs-root/${DESKTOP_FILE} .",
		"sed -i \"s@Exec=.*@Exec=${PREFIX}/bin/${BINNAME}@\" ${DESKTOP_FILE}",

		"cd squashfs-root",
		"if [ ! $PREFIX ]; then",
		"  PREFIX=opt/${APP_PREFIX}",
		"fi",
		"DESTDIR=${dest_dir}",
		"# install icons to linglong package",
		"if [ -d usr/share/icons ]; then",
		"  cd usr",
		"  find share/icons -type f -exec install -D \"{}\" \"${DESTDIR}/${PREFIX}/{}\" \\;",
		"  cd ..",
		"fi",
		"find -type d -exec install -d \"${DESTDIR}/${PREFIX}/lib/${APP_PREFIX}/{}\" \\;",
		"find -type f -exec install -D \"{}\" \"${DESTDIR}/${PREFIX}/lib/${APP_PREFIX}/{}\" \\;",
		"find -type l -exec bash -c \"ln -s \\$(readlink {}) \"${DESTDIR}/${PREFIX}/lib/${APP_PREFIX}/{}\" \" -exec install -D \"{}\" \"${DESTDIR}/${PREFIX}/lib/${APP_PREFIX}/{}\" \\;",
		"cd ..",
		"install -D ${BINNAME} ${DESTDIR}/${PREFIX}/bin/${BINNAME}",
		"install -D ${DESKTOP_FILE} ${DESTDIR}/${PREFIX}/share/applications/${DESKTOP_FILE}",
	}...)

	builder := linglong.LinglongBuilder{
		Package: linglong.Package{
			Appid:       options.packageId,
			Name:        options.packageName,
			Version:     options.packageVersion,
			Kind:        "app",
			Description: options.packageDescription,
		},
		Base: fmt.Sprintf("%s/%s", "org.deepin.base", "25.2.1"),
		Command: []string{
			fmt.Sprintf("/opt/apps/%s/files/bin/%s", options.packageId, options.packageId),
		},
		Build: build,
	}

	if options.appimageFileUrl != "" && options.appimageFileHash != "" {
		var sources []comm.Source
		sources = append(sources, comm.Source{Kind: "file", Digest: options.appimageFileHash, Url: options.appimageFileUrl})
		builder.Sources = sources
	}

	workDir := options.packageId

	if exited, err := fs.CreateDir(workDir); !exited {
		log.Logger.Errorf("create workdir %s: failed: %s", workDir, err)
	}

	// 复制appimage文件到工作目录下的sources目录
	if options.appimageFile != "" {
		data, err := os.ReadFile(options.appimageFile)
		if err != nil {
			return err
		}

		err = os.MkdirAll(comm.LocalPackageSourceDir(workDir), 0755)
		if err != nil {
			return err
		}

		destinationFilePath := filepath.Join(comm.LocalPackageSourceDir(workDir), filepath.Base(options.appimageFile))
		err = os.WriteFile(destinationFilePath, data, 0644)
		if err != nil {
			return err
		}
	}

	linglongYamlPath := filepath.Join(workDir, comm.LinglongYaml)

	// 生成 linglong.yaml 文件
	if builder.CreateLinglongYaml(linglongYamlPath) {
		log.Logger.Infof("generate %s success.", comm.LinglongYaml)
	} else {
		log.Logger.Errorf("generate %s failed", comm.LinglongYaml)
	}

	log.Logger.Info("building linglong package")

	// 构建玲珑包
	if options.buildFlag {
		buildLinglongPath := filepath.Dir(linglongYamlPath)
		builder.LinglongBuild(buildLinglongPath, "ll-builder build --skip-output-check")

		layerOpt := "uab"
		if options.exportLayerFlag {
			layerOpt = "layer"
		}
		builder.LinglongExport(buildLinglongPath, layerOpt)
	}

	return nil
}
