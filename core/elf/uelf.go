/*
 * Copyright (c) 2022. Uniontech Software Ltd. All rights reserved.
 *
 * Author: Heysion Y. <heysion@deepin.com>
 *
 * Maintainer: Heysion Y. <heysion@deepin.com>
 *
 * SPDX-License-Identifier: GNU General Public License v3.0 or later
 */
package elf

import (
	"bytes"
	"fmt"
	. "ll-pica/utils/log"
	"os"
	"path"
	"path/filepath"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	logger = InitLog()
}

var ELF_MAGIC = []byte{0x7f, 0x45, 0x4c, 0x46}

/*
* !@brief IsElfWithPath
* @param elfPath
* @return bool
 */
func IsElfWithPath(elfPath string) bool {
	f, err := os.Open(elfPath)
	if err != nil {
		//logger.Debugf("Open:", elfPath, err)
		return false
	}
	defer f.Close()

	file_header_data := make([]byte, 32)
	n, err := f.Read(file_header_data)
	//logger.Debugf("Read:", elfPath, n, err)
	if err != nil || n <= 30 {
		return false
	}
	return bytes.Equal(file_header_data[:4], ELF_MAGIC)
}

/*!
* @brief GetElfWithPath
* @param dir
* @return []string, error
 */
func GetElfWithPath(dir string) ([]string, error) {

	var real_path = ""
	var elf_paths []string
	if path.IsAbs(dir) {
		real_path = dir
	} else {
		real_path = filepath.Join(os.Getenv("PWD"), dir)
	}
	if real_path[len(real_path)-1:] != "/" {
		real_path = real_path + "/"
	}

	logger.Debugf("GetElfWithPath:", real_path)

	err := filepath.Walk(real_path, func(path string, info os.FileInfo, err error) error {

		if (info != nil && !info.IsDir() && info.Mode().IsRegular()) && IsElfWithPath(path) {
			//fmt.Println("elf: ", path)
			elf_paths = append(elf_paths, path)
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return elf_paths, nil
}
