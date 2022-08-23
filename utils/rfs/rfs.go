/*
 * Copyright (c) 2022. Uniontech Software Ltd. All rights reserved.
 *
 * Author: Heysion Y. <heysion@deepin.com>
 *
 * Maintainer: Heysion Y. <heysion@deepin.com>
 *
 * SPDX-License-Identifier: GNU General Public License v3.0 or later
 */
package rfs

import (
	"fmt"
	. "ll-pica/core/comm"
	. "ll-pica/utils/log"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	logger = InitLog()
}

/*!
 * @brief MountIso 将iso挂载到指定目录
 * @param path 目录路径 ，如/mnt/iso ，iso 挂载文件
 * @return 是否成功
 */
func MountIso(path, iso string) (bool, error) {
	_, msg, err := ExecAndWait(10, "mount", "-o", "loop", iso, path)
	if err != nil {
		logger.Error("mount iso failed!", msg, err)
		return false, err
	}
	return true, nil
}

/*!
 * @brief UmountIso 卸载挂载iso
 * @param path 卸载目录路径
 * @return 是否成功
 */
func UmountIso(path string) (bool, error) {
	logger.Debug("umount iso: ", path)
	if _, msg, err := ExecAndWait(10, "umount", path); err != nil {
		logger.Error("umount iso failed!", msg, err)
		return false, err
	}

	return true, nil
}

/*!
 * @brief MountSquashfs 将squashfs挂载到指定目录
 * @param path 目录路径 ，如/mnt/iso , squashfs 挂载文件
 * @return 是否成功
 */
func MountSquashfs(path, squashfs string) (bool, error) {

	logger.Debugf("mount squashfs %s to %s ", squashfs, path)
	_, msg, err := ExecAndWait(10, "mount", squashfs, path)
	if err != nil {
		logger.Error("mount squashfs failed!", msg, err)
		return false, err
	}
	logger.Debug("mount squashfs success.")
	return true, nil
}

/*!
 * @brief UmountSquashfs 卸载挂载squashfs
 * @param path 卸载目录路径
 * @return 是否成功
 */
func UmountSquashfs(path string) (bool, error) {
	logger.Debug("umount squashfs: ", path)
	if _, msg, err := ExecAndWait(10, "umount", path); err != nil {
		logger.Error("umount squashfs failed!", msg, err)
		return false, err
	}

	return true, nil
}

func MountRfsWithOverlayfs(workdir, rfs, upper, tmpdir, lower string) (bool, error) {
	return MountRfs("overlay", lower, upper, workdir, tmpdir, rfs)
}

/*!
 * @brief MountRfs 使用overlayfs挂载rfs
 * @param rfsPath rfs路径，lower,upper,workdir,tmpdir
 * @return 是否成功
 */
func MountRfs(fstype, lower, upper, workdir, tmpdir, rfsdir string) (bool, error) {

	logger.Debug("mount rfs: ", fstype, lower, upper, workdir, tmpdir, rfsdir)

	switch {
	case fstype == "overlay":
		logger.Debug("SetOverlayfs :", lower, upper, workdir)
		// mount lower dir to upper dir
		msg := fmt.Sprintf("lowerdir=%s:%s,upperdir=%s,workdir=%s", upper, lower, workdir, tmpdir)
		logger.Debug("mount overlayfs flags: ", msg)
		if _, msg, err := ExecAndWait(10, "mount", "-t", "overlay", "overlay", "-o", msg, rfsdir); err != nil {
			logger.Error("mount overlayfs failed: ", msg, err)
			return false, err
		}
		logger.Debug("mount overlayfs success: ", rfsdir)
		return true, nil
	case fstype == "mount":
		logger.Debug("SetMountfs :", lower, upper, workdir)
		logger.Fatal("not support mountfs")
	}
	return false, nil
}

/*!
 * @brief UmountRfs 卸载rfs
 * @param workdir
 * @return 是否成功
 */
func UmountRfs(workdir string) (bool, error) {
	logger.Debug("umountRfs :", workdir)
	// umount upper dir
	_, msg, err := ExecAndWait(10, "umount", workdir)
	if err != nil {
		logger.Error("umount rootfs failed: ", msg, err)
		return false, err
	}
	return true, nil
}
