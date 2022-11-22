/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package elf

import (
	"bytes"
	"fmt"
	. "ll-pica/core/comm"
	. "ll-pica/utils/fs"
	. "ll-pica/utils/log"
	"os"
	"path"
	"path/filepath"
)

var ELF_MAGIC = []byte{0x7f, 0x45, 0x4c, 0x46}

/*
* !@brief IsElfWithPath
* @param elfPath
* @return bool
 */
func IsElfWithPath(elfPath string) bool {
	f, err := os.Open(elfPath)
	if err != nil {
		//Logger.Debugf("Open:", elfPath, err)
		return false
	}
	defer f.Close()

	file_header_data := make([]byte, 32)
	n, err := f.Read(file_header_data)
	//Logger.Debugf("Read:", elfPath, n, err)
	if err != nil || n <= 30 {
		return false
	}
	return bytes.Equal(file_header_data[:4], ELF_MAGIC)
}

// IsElfEntry check with entry is libc_start_main
func IsElfEntry(elfPath string) bool {
	cmd := fmt.Sprintf("nm -D %s | grep -q 'libc_start_main' ", elfPath)
	if ret, msg, err := ExecAndWait(10, "bash", "-c", cmd); err != nil {
		Logger.Debugf("check elf entry failed: %v", err, msg, ret)
		return false
	} else {
		return true
	}
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

	Logger.Debugf("GetElfWithPath:", real_path)

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

// GetElfWithEntry
func GetElfWithEntry(filename string) ([]string, error) {

	var real_path = ""
	var elf_paths []string
	if path.IsAbs(filename) {
		real_path = filename
	} else {
		real_path = filepath.Join(os.Getenv("PWD"), filename)
	}
	if ret, err := CheckFileExits(real_path); err != nil && !ret {
		Logger.Warnf("get elf path failed: %v", err)
		return nil, err
	}

	Logger.Debugf("GetElfWithEntry:", real_path)

	if IsElfEntry(real_path) {
		elf_paths = append(elf_paths, real_path)
	}

	return elf_paths, nil
}
