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
)

// var Logger *zap.SugaredLogger

// func init() {
// 	Logger = InitLog()
// }

/*!
 * @brief MountIso 将iso挂载到指定目录
 * @param path 目录路径 ，如/mnt/iso ，iso 挂载文件
 * @return 是否成功
 */
func MountIso(path, iso string) (bool, error) {
	_, msg, err := ExecAndWait(10, "mount", "-o", "loop", iso, path)
	if err != nil {
		Logger.Error("mount iso failed!", msg, err)
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
	Logger.Debug("umount iso: ", path)
	if _, msg, err := ExecAndWait(10, "umount", path); err != nil {
		Logger.Error("umount iso failed!", msg, err)
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

	Logger.Debugf("mount squashfs %s to %s ", squashfs, path)
	_, msg, err := ExecAndWait(10, "mount", squashfs, path)
	if err != nil {
		Logger.Error("mount squashfs failed!", msg, err)
		return false, err
	}
	Logger.Debug("mount squashfs success.")
	return true, nil
}

/*!
 * @brief UmountSquashfs 卸载挂载squashfs
 * @param path 卸载目录路径
 * @return 是否成功
 */
func UmountSquashfs(path string) (bool, error) {
	Logger.Debug("umount squashfs: ", path)
	if _, msg, err := ExecAndWait(10, "umount", path); err != nil {
		Logger.Error("umount squashfs failed!", msg, err)
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

	Logger.Debug("mount rfs: ", fstype, lower, upper, workdir, tmpdir, rfsdir)

	switch {
	case fstype == "overlay":
		Logger.Debug("SetOverlayfs :", lower, upper, workdir)
		// mount lower dir to upper dir
		msg := fmt.Sprintf("lowerdir=%s:%s,upperdir=%s,workdir=%s", upper, lower, workdir, tmpdir)
		Logger.Debug("mount overlayfs flags: ", msg)
		if _, msg, err := ExecAndWait(10, "mount", "-t", "overlay", "overlay", "-o", msg, rfsdir); err != nil {
			Logger.Error("mount overlayfs failed: ", msg, err)
			return false, err
		}
		Logger.Debug("mount overlayfs success: ", rfsdir)
		return true, nil
	case fstype == "mount":
		Logger.Debug("SetMountfs :", lower, upper, workdir)
		Logger.Fatal("not support mountfs")
	}
	return false, nil
}

/*!
 * @brief UmountRfs 卸载rfs
 * @param workdir
 * @return 是否成功
 */
func UmountRfs(workdir string) (bool, error) {
	Logger.Debug("umountRfs :", workdir)
	// umount upper dir
	_, msg, err := ExecAndWait(10, "umount", workdir)
	if err != nil {
		Logger.Error("umount rootfs failed: ", msg, err)
		return false, err
	}
	return true, nil
}
