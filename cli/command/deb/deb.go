/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package deb

import (
	"github.com/spf13/cobra"
	"pkg.deepin.com/linglong/pica/cli/command/adep"
	"pkg.deepin.com/linglong/pica/cli/command/convert"
	minit "pkg.deepin.com/linglong/pica/cli/command/init"
)

func NewDebCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deb",
		Short: "Convert deb package to linglong package",
	}

	cmd.AddCommand(minit.NewInitCommand())
	cmd.AddCommand(convert.NewConvertCommand())
	cmd.AddCommand(adep.NewADepCommand())
	return cmd
}
