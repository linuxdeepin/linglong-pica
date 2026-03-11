/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package completion

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewCompletionCommand(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for ll-pica.

Examples:
  ll-pica completion bash > ll-pica.bash
  ll-pica completion zsh > _ll-pica
  ll-pica completion fish > ll-pica.fish`,
		Args: cobra.ExactArgs(1),
		ValidArgs: []string{
			"bash",
			"zsh",
			"fish",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			switch shell {
			case "bash":
				return rootCmd.GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			default:
				return fmt.Errorf("unsupported shell %q", shell)
			}
		},
	}

	cmd.SetHelpCommand(&cobra.Command{Hidden: true})
	return cmd
}
