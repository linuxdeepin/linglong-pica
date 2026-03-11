/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAddCommands(t *testing.T) {
	root := &cobra.Command{Use: "ll-pica"}
	AddCommands(root)

	debCmd, _, err := root.Find([]string{"deb"})
	if err != nil || debCmd == nil || debCmd.Name() != "deb" {
		t.Fatalf("expected deb command, got err=%v", err)
	}

	appimageCmd, _, err := root.Find([]string{"appimage"})
	if err != nil || appimageCmd == nil || appimageCmd.Name() != "appimage" {
		t.Fatalf("expected appimage command, got err=%v", err)
	}

	flatpakCmd, _, err := root.Find([]string{"flatpak"})
	if err != nil || flatpakCmd == nil || flatpakCmd.Name() != "flatpak" {
		t.Fatalf("expected flatpak command, got err=%v", err)
	}

	completionCmd, _, err := root.Find([]string{"completion"})
	if err != nil || completionCmd == nil || completionCmd.Name() != "completion" {
		t.Fatalf("expected completion command, got err=%v", err)
	}

	if c, _, err := root.Find([]string{"deb", "init"}); err != nil || c == nil || c.Name() != "init" {
		t.Fatalf("expected deb init command, got err=%v", err)
	}
	if c, _, err := root.Find([]string{"deb", "convert"}); err != nil || c == nil || c.Name() != "convert" {
		t.Fatalf("expected deb convert command, got err=%v", err)
	}
	if c, _, err := root.Find([]string{"deb", "adep"}); err != nil || c == nil || c.Name() != "adep" {
		t.Fatalf("expected deb adep command, got err=%v", err)
	}
	if c, _, err := root.Find([]string{"appimage", "convert"}); err != nil || c == nil || c.Name() != "convert" {
		t.Fatalf("expected appimage convert command, got err=%v", err)
	}
	if c, _, err := root.Find([]string{"flatpak", "convert"}); err != nil || c == nil || c.Name() != "convert" {
		t.Fatalf("expected flatpak convert command, got err=%v", err)
	}
}
