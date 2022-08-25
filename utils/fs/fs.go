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
	"fmt"
	"io"
	"io/ioutil"
	. "ll-pica/utils/log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	logger = InitLog()
}

/*!
 * @brief 检查是否是目录
 * @param dir 目录路径
 * @return 是否是目录
 */
func IsDir(file string) bool {
	if info, err := os.Stat(file); err == nil {
		return info.IsDir()
	}
	return false
}

/*!
 * @brief 检查文件or目录是否存在
 * @param file 文件路径
 * @return 是否存在
 */
func CheckFileExits(file string) (bool, error) {

	logger.Debug("check file exists: ", file)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		logger.Error("file not exists and exit", err)
		return false, err
	} else if err == nil {
		logger.Debug("file exists")
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

	logger.Debug("create file: ", file)
	if err := os.MkdirAll(file, 0755); err == nil {
		logger.Debug("create file: ", file, " mask: 0755")
		return true, nil
	} else {
		logger.Error("create file error: ", err)
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

/*!
 * @brief 移动目录或者文件,并会创建目标路径,如果目标存在，则不会移动覆盖（文件目录权限不变，链接文件保持）
 * @param src 源文件或者目录
 * @param dst 目标文件或者目录
 * @return 是否成功
 */
func MoveFileOrDir(src, dst string) (bool, error) {
	if ret, err := CheckFileExits(src); !ret {
		logger.Warnw(src, " no existd!")
		return false, err
	}
	dstDirPath := GetFilePPath(dst)
	CreateDir(dstDirPath)
	//转换绝对路径
	src, _ = filepath.Abs(src)
	dst, _ = filepath.Abs(dst)
	if err := os.Rename(src, dst); err != nil {
		return false, err
	}
	return true, nil
}

/*!
 * @brief 拷贝文件包括文件权限，并会创建目标路径(链接文件无法保持)
 * @param src 源文件
 * @param dst 目标文件
 * @return 是否成功
 */
func CopyFile(src, dst string) (bool, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return false, err
	}
	defer srcFile.Close()

	dstDirPath := GetFilePPath(dst)
	CreateDir(dstDirPath)

	//获取源文件的权限
	fi, _ := srcFile.Stat()
	perm := fi.Mode()

	//desFile, err := os.Create(des)  //无法复制源文件的所有权限
	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm) //复制源文件的所有权限
	if err != nil {
		return false, err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return false, err
	}
	return true, nil
}

/*!
 * @brief 拷贝目录包括文件权限，并会创建目标路径(链接文件无法保持)
 * @param src 源目录路径
 * @param dst 目标目录路径
 * @return 是否成功
 */
func CopyDir(src, dst string) bool {
	//检查源目录是否存在
	if ret, _ := CheckFileExits(src); !ret {
		logger.Warnw(src, " no existd!")
		return false
	}

	if strings.TrimSpace(src) == strings.TrimSpace(dst) {
		logger.Warnw("源路径与目标路径一样")
		return false
	}

	//转化为绝对路径
	src, _ = filepath.Abs(src)
	dst, _ = filepath.Abs(dst)

	//创建目录路径
	CreateDir(dst)

	err := filepath.Walk(src, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}

		//复制目录是将源目录中的子目录复制到目标路径中，不包含源目录本身
		if path == src {
			return nil
		}

		//生成新路径
		destNewPath := strings.Replace(path, src, dst, 1)
		fmt.Printf("path: %s\n", path)
		fmt.Printf("destNewPath: %s\n", destNewPath)

		if !f.IsDir() {
			CopyFile(path, destNewPath)
		} else {
			if ret, _ := CheckFileExits(src); !ret {
				CreateDir(destNewPath)
				return nil
			}
		}

		return nil
	})

	return err == nil
}

// Copy File Keep Permission
// return error if copy fails
func CopyFileKeepPermission(src, dst string, mod, owner bool) (err error) {
	srcFd, err := os.Open(src)
	if err != nil {
		return
	}
	defer srcFd.Close()

	dstFd, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if e := dstFd.Close(); e != nil {
			err = e
		}
	}()

	if _, err = io.Copy(dstFd, srcFd); err != nil {
		return err
	}

	if err = dstFd.Sync(); err != nil {
		return err
	}

	fileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if mod {
		if err = os.Chmod(dst, fileStat.Mode()); err != nil {
			return err
		}
	}

	if owner {
		if stat, ok := fileStat.Sys().(*syscall.Stat_t); ok {
			UID := int(stat.Uid)
			GID := int(stat.Gid)
			if err = os.Chown(dst, UID, GID); err != nil {
				return err
			}
		}
	}

	return nil
}

/*!
 * @brief CopyDirKeepPathAndPerm 复制文件同时保留目录结构
 * @param src path 来源
 * @param dst path 目的路径
 * @return 成功与失败
 */
func CopyDirKeepPathAndPerm(src string, dst string, force, mod, owner bool) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		logger.Errorf("source only support directory")
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && !os.IsNotExist(err) {
		if !force {
			logger.Errorf("destination already exists failed")
			return fmt.Errorf("destination already exists")
		} else {
			if err := os.RemoveAll(dst); err != nil {
				return err
			}
		}
	}

	if err = os.MkdirAll(dst, si.Mode()); err != nil {
		return err
	}

	items, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	for _, item := range items {
		srcPath := filepath.Join(src, item.Name())
		dstPath := filepath.Join(dst, item.Name())

		if item.IsDir() {
			// not drop subdirectories
			if err = CopyDirKeepPathAndPerm(srcPath, dstPath, false, mod, owner); err != nil {
				return err
			}
		} else {
			// copy link data
			// fixme(heysion)
			if item.Mode()&os.ModeSymlink != 0 {
				realPath, err := os.Readlink(srcPath)
				if err != nil {
					logger.Warnf("link failed to read link data: %v %s %s", err, realPath, srcPath)
					return err
				}
				if realPathFileInfo, err := os.Stat(dst); err != nil {
					if realPathFileInfo.IsDir() {
						if err = CopyDirKeepPathAndPerm(realPath, dstPath, false, mod, owner); err != nil {
							return err
						}
					} else {
						// copy data drop the mod and owner
						if err = CopyFileKeepPermission(srcPath, dstPath, false, false); err != nil {
							logger.Warnf("link %s to %s failed: %v", srcPath, dstPath, err)
							return err
						}
					}
				}

				continue
			}
			// copy file keep the mod and owner
			if err = CopyFileKeepPermission(srcPath, dstPath, mod, owner); err != nil {
				return err
			}
		}
	}

	return nil
}
