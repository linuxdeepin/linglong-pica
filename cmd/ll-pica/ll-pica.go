/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package main

import (
	"os"

	"github.com/spf13/cobra"
	"pkg.deepin.com/linglong/pica/cli"
	"pkg.deepin.com/linglong/pica/cli/command/commands"
	"pkg.deepin.com/linglong/pica/cli/version"
	"pkg.deepin.com/linglong/pica/tools/log"
)

func main() {
	log.Logger = log.InitLog()
	defer log.Logger.Sync()

	if err := runPica(); err != nil {
		log.Logger.Errorf("run pica failed: %v", err)
		os.Exit(1)
	}
}

func newPicaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ll-pica",
		Short: "Convert deb, appimage and flatpak package to linglong package",
		Long: `Convert packages to uab. For example:
Simple:
	ll-pica deb init -c package -w work-dir
	ll-pica deb convert -c package.yaml -w work-dir
	ll-pica appimage convert -f xxx.appimage -i io.github.demo -v 1.0.0.0
	ll-pica flatpak convert org.kde.kate --build
	ll-pica help
		`,
		Version: version.Version,
	}

	cmd.CompletionOptions.DisableDefaultCmd = true
	cli.SetupRootCommand(cmd)
	commands.AddCommands(cmd)
	return cmd
}

func runPica() error {
	cmd := newPicaCommand()
	return cmd.Execute()
}
