/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package commands

import (
	"github.com/spf13/cobra"
	"pkg.deepin.com/linglong/pica/cli/command/appimage"
	"pkg.deepin.com/linglong/pica/cli/command/completion"
	"pkg.deepin.com/linglong/pica/cli/command/deb"
	"pkg.deepin.com/linglong/pica/cli/command/flatpak"
)

func AddCommands(cmd *cobra.Command) {
	cmd.AddCommand(deb.NewDebCommand())
	cmd.AddCommand(appimage.NewAppimageCommand())
	cmd.AddCommand(flatpak.NewFlatpakCommand())
	cmd.AddCommand(completion.NewCompletionCommand(cmd))
}
