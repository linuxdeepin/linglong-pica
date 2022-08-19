package fs

import (
	"io"
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

/*!
 * @brief 移动目录或者文件,并会创建目标路径（文件目录权限不变，链接文件保持）
 * @param src 源文件或者目录
 * @param dst 目标文件或者目录
 * @return 是否成功
 */
func MoveFileOrDir(src, dst string) (bool, error) {
	if ret, err := CheckFileExits(src); !ret {
		_logger.Warnw(src, " no existd!")
		return false, err
	}
	dstDirPath := GetFilePPath(dst)
	CreateDir(dstDirPath)
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
