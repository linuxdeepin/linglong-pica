/*
 * Copyright (c) 2022. Uniontech Software Ltd. All rights reserved.
 *
 * Author: Heysion Y. <heysion@deepin.com>
 *
 * Maintainer: Heysion Y. <heysion@deepin.com>
 *
 * SPDX-License-Identifier: GNU General Public License v3.0 or later
 */

package fs

import (
	. "ll-pica/utils/log"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

var _logger *zap.SugaredLogger

func init() {
	_logger = InitLog()
}

/*!
 * @brief 检查文件or目录是否存在
 * @param file 文件路径
 * @return 是否存在
 */
func CheckFileExits(file string) (bool, error) {

	_logger.Debug("check file exists: ", file)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		_logger.Error("file not exists and exit", err)
		return false, err
	} else if err == nil {
		_logger.Debug("file exists")
		return true, nil
	}
	return false, nil
}

/*!
 * @brief 创建目录
 * @param file 目录路径
 * @return 是否存在
 */
func CreateDir(file string) (bool, error) {

	_logger.Debug("create file: ", file)
	if err := os.MkdirAll(file, 0755); err == nil {
		_logger.Debug("create file: ", file, " mask: 0755")
		return true, nil
	} else {
		_logger.Error("create file error: ", err)
		return false, err
	}

}

/*!
 * @brief 获取文件名
 * @param file 文件
 * @return 是否存在
 */
func GetFileName(file string) string {

	return filepath.Base(file)

}

/*!
 * @brief 获取文件的目录
 * @param file 文件
 * @return 是否存在
 */
func GetFilePPath(file string) string {

	return filepath.Dir(file)

}
