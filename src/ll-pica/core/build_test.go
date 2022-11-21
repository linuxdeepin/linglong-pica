/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package core

import (
	"fmt"
	. "ll-pica/utils/fs"
	"testing"
)

var binReactor = new(BinFormatReactor)

func init() {
	binReactor.SearchPath = "/mnt/workdir/basedir"
	fmt.Printf("init: %v\n", binReactor)
}

func TestGetElfList(t *testing.T) {
	binReactor.GetElfList(binReactor.SearchPath + "/lib")
	fmt.Printf("TestGetElfList: %d\n", len(binReactor.ElfLDDPath))
	if len(binReactor.ElfLDDPath) > 0 {
		t.Logf("success")
		return
	} else {
		t.Fatalf("failed %+v %d", binReactor, len(binReactor.ElfLDDPath))
	}

}

func TestRenderElfWithLDD(t *testing.T) {

	elfLDDLog := "/mnt/workdir/elfldd.log"
	elfLDDShell := "/mnt/workdir/elfldd.sh"

	binReactor.RenderElfWithLDD(elfLDDLog, elfLDDShell)
	if ret, _ := CheckFileExits(elfLDDShell); !ret {
		t.Errorf("%s %s", elfLDDLog, elfLDDShell)
		return
	}

	t.Logf("success")

}

func TestFixElfLDDPath(t *testing.T) {
	excludeList := []string{binReactor.SearchPath}
	binReactor.FixElfLDDPath(excludeList)
	fmt.Printf("TestFixElfLDDPath: %d\n", len(binReactor.ElfLDDPath))
	if len(binReactor.ElfLDDPath) > 0 {
		t.Fatalf("failed %d", len(binReactor.ElfLDDPath))
	}
}
