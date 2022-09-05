/*
 * Copyright (c) 2022. Uniontech Software Ltd. All rights reserved.
 *
 * Author: Heysion Y. <heysion@deepin.com>
 *
 * Maintainer: Heysion Y. <heysion@deepin.com>
 *
 * SPDX-License-Identifier: GNU General Public License v3.0 or later
 */
package comm

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
)

// ExecAndWait
func TestExecAndWait(t *testing.T) {
	if v1, v2, err := ExecAndWait(10, "ls", "-al"); err != nil {
		t.Error(err, v1, v2)
	}
}

// GetFileSha256
func TestGetFileSha256(t *testing.T) {
	oneFileSha256 := "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
	file, err := ioutil.TempFile("/tmp/", "sha256_")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	file.WriteString("1")

	file.Close()

	if fileSha256, err := GetFileSha256(file.Name()); fileSha256 != oneFileSha256 || err != nil {
		t.Error("failed: ", err, file.Name())
	}
}
