/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package convert

import (
	"strings"
	"testing"
)

func TestRunConvertValidation(t *testing.T) {
	tests := []struct {
		name    string
		opts    convertOptions
		wantErr string
	}{
		{
			name:    "missing package id",
			opts:    convertOptions{packageVersion: "1.0.0.0", appimageFile: "demo.AppImage"},
			wantErr: "package id is required",
		},
		{
			name:    "missing version",
			opts:    convertOptions{packageId: "io.demo", appimageFile: "demo.AppImage"},
			wantErr: "package version is required",
		},
		{
			name:    "invalid package id",
			opts:    convertOptions{packageId: "../io.demo", packageVersion: "1.0.0.0", appimageFile: "demo.AppImage"},
			wantErr: "invalid package id",
		},
		{
			name:    "missing file and url",
			opts:    convertOptions{packageId: "io.demo", packageVersion: "1.0.0.0"},
			wantErr: "file option or url option is required",
		},
		{
			name:    "url without hash",
			opts:    convertOptions{packageId: "io.demo", packageVersion: "1.0.0.0", appimageFileUrl: "https://example.com/a.AppImage"},
			wantErr: "hash option is required when use url option",
		},
		{
			name:    "invalid suffix",
			opts:    convertOptions{packageId: "io.demo", packageVersion: "1.0.0.0", appimageFile: "demo.deb"},
			wantErr: "appimage file must be .AppImage or .appimage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runConvert(&tt.opts)
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error contains %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}
