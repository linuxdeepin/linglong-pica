/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package commands

import (
	"github.com/spf13/cobra"
	"pkg.deepin.com/linglong/pica/cli/command/convert"
	minit "pkg.deepin.com/linglong/pica/cli/command/init"
)

func AddCommands(cmd *cobra.Command) {
	cmd.AddCommand(minit.NewInitCommand())
	cmd.AddCommand(convert.NewConvertCommand())
}
