/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package appimage

import (
	"github.com/spf13/cobra"
	appconvert "pkg.deepin.com/linglong/pica/cli/appimage/convert"
)

func NewAppimageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "appimage",
		Short: "Convert appimage package to linglong package",
	}

	cmd.AddCommand(appconvert.NewConvertCommand())
	return cmd
}
