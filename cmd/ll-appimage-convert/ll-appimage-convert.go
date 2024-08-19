/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package main

import (
	"os"

	"github.com/spf13/cobra"

	"pkg.deepin.com/linglong/pica/cli"
	"pkg.deepin.com/linglong/pica/cli/appimage/convert"
	"pkg.deepin.com/linglong/pica/tools/log"
)

func main() {
	log.Logger = log.InitLog()
	defer log.Logger.Sync()

	if err := run(); err != nil {
		log.Logger.Errorf("run pica failed: %v", err)
		os.Exit(1)
	}
}

func newAppimageConvertCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ll-appimage-convert",
		Short: "appimage package convert linglong package",
		Long: `Convert the appimage to uab. For example:
 Simple:
	 ll-appimage-convert  convert -f xxx.appimage -i "io.github.demo" -n "io.github.demo" -v "1.0.0.0" -d "this is a appimage convert demo" -b
	 ll-appimage-convert help
		 `,
		Version: "1.0.0",
	}

	cmd.CompletionOptions.DisableDefaultCmd = true
	cli.AppimageConvertSetupRootCommand(cmd)
	cmd.AddCommand(convert.NewConvertCommand())
	return cmd
}

func run() error {
	cmd := newAppimageConvertCommand()
	return cmd.Execute()
}
