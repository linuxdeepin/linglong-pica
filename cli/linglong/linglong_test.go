/*
 * SPDX-FileCopyrightText: 2026 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package linglong

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pkg.deepin.com/linglong/pica/cli/comm"
)

func TestCreateLinglongYamlIncludesBuildextAndSingleSource(t *testing.T) {
	output := filepath.Join(t.TempDir(), "linglong.yaml")
	builder := LinglongBuilder{
		Package: Package{
			Appid:       "io.demo.app",
			Name:        "demo",
			Version:     "1.0.0.0",
			Kind:        "app",
			Description: "demo package",
		},
		Base:    "org.deepin.foundation/25.2.0",
		Runtime: "org.deepin.Runtime/25.2.0",
		Command: []string{"/opt/apps/io.demo.app/files/bin/demo"},
		Sources: []comm.Source{
			{Kind: "file", Url: "https://example.com/demo.deb", Digest: "deadbeef"},
		},
		Build: []string{"install -d $PREFIX/bin"},
		BuildExt: BuildExt{
			Apt: AptExt{
				BuildDepends: []string{"cmake", "ninja-build"},
				Depends:      []string{"libc6", "libqt5core5a"},
			},
		},
	}

	if !builder.CreateLinglongYaml(output) {
		t.Fatalf("CreateLinglongYaml returned false")
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output failed: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"sources:",
		"url: https://example.com/demo.deb",
		"digest: deadbeef",
		"buildext:",
		"build_depends:",
		"- cmake",
		"depends:",
		"- libc6",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, content)
		}
	}
}

func TestCreateLinglongYamlKeepsRuntimeEmptyWithComment(t *testing.T) {
	output := filepath.Join(t.TempDir(), "linglong.yaml")
	builder := LinglongBuilder{
		Package: Package{
			Appid:       "io.demo.app",
			Name:        "demo",
			Version:     "1.0.0.0",
			Kind:        "app",
			Description: "demo package",
		},
		Base:    "org.deepin.base/25.2.1",
		Runtime: "",
		Command: []string{"/opt/apps/io.demo.app/files/bin/demo"},
		Build:   []string{"install -d $PREFIX/bin"},
	}

	if !builder.CreateLinglongYaml(output) {
		t.Fatalf("CreateLinglongYaml returned false")
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output failed: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"# runtime is only needed for Qt/Dtk-based projects; leave it empty by default.",
		"runtime: ",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, content)
		}
	}
	if strings.Contains(content, "runtime: org.deepin") {
		t.Fatalf("expected runtime to stay empty, got:\n%s", content)
	}
}
