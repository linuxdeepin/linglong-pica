/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package cli

import (
	"github.com/spf13/cobra"
	cliflags "pkg.deepin.com/linglong/pica/cli/flags"
	"pkg.deepin.com/linglong/pica/tools/log"
)

var (
	disableDevelop string
)

func setupCommonRootCommand(rootCmd *cobra.Command) (*cliflags.CliOptions, *cobra.Command) {
	opts := cliflags.NewCliOptions()
	opts.InstallFlags(rootCmd.Flags())
	rootCmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "V", false, "verbose output")
	// go build -ldflags '-X pkg.deepin.com/linglong/pica/cmd/ll-pica/utils/log.disableLogDebug=yes -X main.disableDevelop=yes'
	// fmt.Printf("disableDevelop: %s\n", disableDevelop)
	if disableDevelop != "" {
		log.Logger.Debugf("develop mode disable")
		opts.Debug = false
	} else {
		log.Logger.Debugf("develop mode enabled")
		opts.Debug = true
		// debug mode enable verbose mode
		opts.Verbose = true
	}
	return opts, nil
}

func SetupRootCommand(rootCmd *cobra.Command) (opts *cliflags.CliOptions, helpCmd *cobra.Command) {
	rootCmd.SetVersionTemplate("pica version {{.Version}}\n")
	return setupCommonRootCommand(rootCmd)
}
