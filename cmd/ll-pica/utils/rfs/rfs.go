/*
 * SPDX-FileCopyrightText: 2022 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package rfs

import (
	"fmt"

	"pkg.deepin.com/linglong/pica/cmd/ll-pica/core/comm"
	"pkg.deepin.com/linglong/pica/cmd/ll-pica/utils/log"
)

/*!
 * @brief MountIso 将iso挂载到指定目录
 * @param path 目录路径 ，如/mnt/iso ，iso 挂载文件
 * @return 是否成功
 */
func MountIso(path, iso string) (bool, error) {
	_, msg, err := comm.ExecAndWait(10, "mount", "-o", "loop", iso, path)
	if err != nil {
		log.Logger.Warnf("mount iso failed!", msg, err)
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
	log.Logger.Debug("umount iso: ", path)
	if _, msg, err := comm.ExecAndWait(10, "umount", path); err != nil {
		log.Logger.Error("umount iso failed!", msg, err)
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

	log.Logger.Debugf("mount squashfs %s to %s ", squashfs, path)
	_, msg, err := comm.ExecAndWait(10, "mount", squashfs, path)
	if err != nil {
		log.Logger.Error("mount squashfs failed!", msg, err)
		return false, err
	}
	log.Logger.Debug("mount squashfs success.")
	return true, nil
}

/*!
 * @brief UmountSquashfs 卸载挂载squashfs
 * @param path 卸载目录路径
 * @return 是否成功
 */
func UmountSquashfs(path string) (bool, error) {
	log.Logger.Debug("umount squashfs: ", path)
	if _, msg, err := comm.ExecAndWait(10, "umount", path); err != nil {
		log.Logger.Error("umount squashfs failed!", msg, err)
		return false, err
	}

	return true, nil
}

func MountRfsWithOverlayfs(lowerRuntimeDir, lowerFilesSystem, lowerInitDir, upper, workdir, rootfs string) (bool, error) {
	// lowerRuntimeDir , runtimedir/files have bug for first lowdir that can not chroot .
	// fixme:(heysion)
	// MountRfs("overlay", lowerRuntimeDir, lowerFilesSystem, lowerInitDir, upper, workdir, rootfs)
	return MountRfs("overlay", lowerInitDir, lowerFilesSystem, lowerRuntimeDir, upper, workdir, rootfs)
}

/*!
 * @brief MountRfs 使用overlayfs挂载rfs
 * @param rfsPath rfs路径，lower,upper,workdir,tmpdir
 * @return 是否成功
 */
func MountRfs(fstype, lowerTop, lowerMid, lowerBottom, upper, workdir, rootfs string) (bool, error) {

	log.Logger.Debugf("mount rfs: ", fstype, lowerTop, lowerMid, lowerBottom, upper, workdir, rootfs)

	switch {
	case fstype == "overlay":
		log.Logger.Debug("SetOverlayfs :", lowerTop, lowerMid, lowerBottom, upper, rootfs)
		// mount lower dir to upper dir
		msg := fmt.Sprintf("lowerdir=%s:%s:%s,upperdir=%s,workdir=%s", lowerTop, lowerMid, lowerBottom, upper, workdir)
		log.Logger.Debug("mount overlayfs flags: ", msg)
		if _, msg, err := comm.ExecAndWait(10, "mount", "-t", "overlay", "overlay", "-o", msg, rootfs); err != nil {
			log.Logger.Error("mount overlayfs failed: ", msg, err)
			return false, err
		}
		log.Logger.Debug("mount overlayfs success: ", rootfs)
		return true, nil
	case fstype == "mount":
		log.Logger.Debug("SetMountfs :", lowerTop, upper, workdir)
		log.Logger.Fatal("not support mountfs")
	}
	return false, nil
}

/*!
 * @brief UmountRfs 卸载rfs
 * @param workdir
 * @return 是否成功
 */
func UmountRfs(workdir string) (bool, error) {
	log.Logger.Debug("umountRfs :", workdir)
	// umount upper dir

	if ret, msg, err := comm.ExecAndWait(10, "umount", workdir); err != nil {
		log.Logger.Warnf("umount rootfs failed: ", workdir, msg, err, ret)
		if ret, msg, err := comm.ExecAndWait(10, "umount", "-R", workdir); err == nil {
			return true, nil
		} else {
			log.Logger.Warnf("umount -R rootfs failed: ", workdir, msg, err, ret)
		}
		return false, err
	}
	return true, nil
}
