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
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	. "ll-pica/utils/log"
	"os"
	"path/filepath"
	"regexp"
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

	logger.Debugf("check file exists: ", file)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		logger.Warnf("file not exists and exit", err)
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
 * @brief RemovePath 删除指定路径
 * @param file 目录路径或者文件路径
 * @return (bool, error)
 */
func RemovePath(file string) (bool, error) {

	logger.Debugf("remove path: %s", file)
	if ret, _ := CheckFileExits(file); ret {
		if err := os.RemoveAll(file); err == nil {
			logger.Debugf("remove path: %s", file)
			return true, nil
		} else {
			logger.Debugf("remove path error: ", err)
			return false, err
		}
	}
	return false, errors.New("Error: file not exists!")
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
 * @brief 移动目录或者文件,并会创建目标路径,如果目标文件存在，则会移动覆盖（文件目录权限不变，链接文件保持）
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
		logger.Warnf("source only support directory")
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && !os.IsNotExist(err) {
		if !force {
			logger.Warnf("destination already exists failed")
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
				if realLink, err := os.Readlink(srcPath); err == nil && (realLink == "." || realLink == "..") {
					// skip special link files
					continue
				} else if err != nil {
					// skip  can not read link
					continue
				}
				realPath, err := filepath.Abs(srcPath)
				if err != nil {
					logger.Warnf("link failed to read link data: %v %s %s", err, realPath, srcPath)
					return err
				}
				if realPathFileInfo, err := os.Stat(realPath); err != nil {
					// broken link
					logger.Warnf("link failed to stat link data: %v %s %s", err, realPath, srcPath)
					continue
				} else {
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

// FindBundlePath
func FindBundlePath(flie string) ([]string, error) {
	if ret, _ := CheckFileExits(flie); ret {

		bundleList := []string{}

		err := filepath.Walk(flie, func(path string, info os.FileInfo, err error) error {

			if (info != nil && !info.IsDir() && info.Mode().IsRegular()) && HasBundleName(info.Name()) {
				//fmt.Println("elf: ", path)
				bundleList = append(bundleList, path)
			}
			return nil
		})
		if err != nil {
			logger.Debugf("get bundle file failed: %v", err)
			return nil, err
		}
		if len(bundleList) > 0 {
			return bundleList, nil
		}

	}
	return nil, fmt.Errorf("not found: %s", flie)
}

// HasBundleName
func HasBundleName(name string) bool {
	return strings.HasSuffix(name, ".uab")
}

// 初始化desktop文件
type DesktopData map[string]map[string]string

const (
	GroupBegin uint32 = iota
	KeyValue
	Empty
	Comments
)

func DesktopInit(desktopFilePath string) (bool, DesktopData) {
	if ret, err := CheckFileExits(desktopFilePath); !ret && err != nil {
		logger.Errorw("desktop file not exists：", desktopFilePath)
		return false, nil
	}
	file, err := os.Open(desktopFilePath)
	if err != nil {
		logger.Errorw("open file failed: ", desktopFilePath)
		return false, nil
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	lineType := func(line string) uint32 {
		for _, c := range line {
			switch c {
			case '#':
				return Comments
			case '[':
				return GroupBegin
			case ' ':
				break
			case '=':
				return KeyValue
			case '\n':
				break
			default:
				return KeyValue
			}
		}
		return Empty
	}

	parseGroupKey := func(line string) string {
		newLine := strings.Replace(line, "[", "", -1)
		newLine = strings.Replace(newLine, "]", "", -1)
		return newLine
	}

	parseKeyValue := func(line string) (string, string) {
		value := strings.Split(line, "=")
		return value[0], value[1]
	}
	data := make(DesktopData, 10)
	var groupName string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				logger.Debug("File read ok! : ", desktopFilePath)
				break
			} else {
				logger.Errorw("Read file error! : ", desktopFilePath)
				return false, nil
			}
		}
		// 去掉换行符号
		line = strings.TrimRight(line, "\r\n")
		switch lineType(line) {
		case GroupBegin:
			groupName = parseGroupKey(line)
			break
		case KeyValue:
			key, value := parseKeyValue(line)
			if data[groupName] == nil {
				subMap := make(map[string]string, 200)
				data[groupName] = subMap
			}

			data[groupName][key] = value

			break
		case Empty:
		case Comments:
			break
		}
	}
	//fmt.Printf("%v", data)
	return true, data
}

// 通过desktop文件返回其groupName
func DesktopGroupname(desktopFile string) []string {
	ok, data := DesktopInit(desktopFile)
	if !ok && data == nil {
		logger.Errorw("Init dekstop failed! : ", desktopFile)
		return nil
	}
	groupNmaeList := []string{}
	for name := range data {
		groupNmaeList = append(groupNmaeList, name)
		fmt.Printf("%s\n", data[name]["Exec"])
	}
	return groupNmaeList
}

// 转换Exec字段为玲珑格式
func TransExecToLl(exec, appid string) string {
	// 去掉/usr/bin/
	if ret := strings.HasPrefix(exec, "/usr/bin"); ret {
		exec = strings.Replace(exec, "/usr/bin/", "", -1)
	}
	// 去掉首尾空格
	exec = strings.TrimSpace(exec)
	// 添加 ll-cli run appid --exec
	exec = "ll-cli run " + appid + " --exec " + "\"" + exec + "\""

	// 查找占位符
	findSign := func(exec string) string {
		reg := regexp.MustCompile("%.")
		if ret := reg.FindAllString(exec, -1); ret != nil {
			return ret[0]
		}
		return ""
	}

	// 占位符放到引号外
	if ret := findSign(exec); ret != "" {
		exec = strings.Replace(exec, " "+ret, "", -1)
		exec = exec + " " + ret
	}
	return exec
}

// 转换icon字段为玲珑格式
func TransIconToLl(iconValue string) string {
	// 去掉首尾空格
	iconValue = strings.TrimSpace(iconValue)
	// 如果icon以/usr开头
	if strings.HasPrefix(iconValue, "/usr/share") && strings.HasSuffix(iconValue, ".svg") {
		iconValue = GetFileName(iconValue)
		iconValue = strings.Replace(iconValue, ".svg", "", -1)
	}
	if strings.HasPrefix(iconValue, "/usr/share") && strings.HasSuffix(iconValue, ".png") {
		iconValue = GetFileName(iconValue)
		iconValue = strings.Replace(iconValue, ".png", "", -1)
	}

	return iconValue
}
