/*
 * Copyright (c) 2022. Uniontech Software Ltd. All rights reserved.
 *
 * Author: Jianqiang Liu <liujianqiang@deepin.com>
 *
 * Maintainer: Jianqiang Liu <liujianqiang@deepin.com>
 *
 * SPDX-License-Identifier: GNU General Public License v3.0 or later
 */

package info

import (
	"encoding/json"
	. "ll-pica/core/comm"
	. "ll-pica/utils/fs"
	. "ll-pica/utils/log"
	"os"
	"path/filepath"
	"runtime"

	"go.uber.org/zap"
)

type InfoApp struct {
	Appid          string          `json:"appid"`
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	Kind           string          `json:"kind"`
	Description    string          `json:"description"`
	Runtime        string          `json:"runtime"`
	Arch           []string        `json:"arch"`
	Permissions    InfoPermissions `json:"permissions"`
	SupportPlugins []string        `json:"support-plugins"`
	Plugins        []string        `json:"plugins"`
}

type InfoPermissions struct {
	AutoStart     bool `json:"autostart"`
	Notification  bool `json:"notification"`
	Trayicon      bool `json:"trayicon"`
	Clipboard     bool `json:"clipboard"`
	Account       bool `json:"account"`
	Bluetooth     bool `json:"bluetooth"`
	Camera        bool `json:"camera"`
	AudioRecord   bool `json:"audio_record"`
	InstalledApps bool `json:"installed_apps"`
}

var logger *zap.SugaredLogger

func init() {
	logger = InitLog()
}

func CreateInfo(infoDir string, debInfo DebConfig) (bool, error) {
	if ret, err := CheckFileExits(infoDir); !ret && err != nil {
		logger.Errorw("info.json dir not exists! : ", infoDir)
		return false, err
	}
	infoFilePath := filepath.Clean(infoDir) + "/info.json"

	infoApp := &InfoApp{
		Appid:       debInfo.Info.Appid,
		Name:        debInfo.Info.Name,
		Version:     debInfo.Info.Version,
		Kind:        "app",
		Description: debInfo.Info.Description,
		Runtime:     "org.deepin.Runtime/20.5.0/x86_64",
	}
	infoApp.Permissions.AutoStart = false
	infoApp.Permissions.Notification = false
	infoApp.Permissions.Trayicon = false
	infoApp.Permissions.Clipboard = false
	infoApp.Permissions.Account = false
	infoApp.Permissions.Bluetooth = false
	infoApp.Permissions.Camera = false
	infoApp.Permissions.AudioRecord = false
	infoApp.Permissions.InstalledApps = false

	hostArch := runtime.GOARCH
	if hostArch == "amd64" {
		hostArch = "x86_64"
	}
	infoApp.Arch = append(infoApp.Arch, hostArch)
	infoApp.SupportPlugins = make([]string, 0)
	infoApp.Plugins = make([]string, 0)

	data, err := json.MarshalIndent(infoApp, "", "\t")
	if err != nil {
		logger.Errorw("序列化错误： ", infoFilePath)
		return false, err
	}

	// 创建文件
	file, err := os.Create(infoFilePath)

	if err != nil {
		logger.Errorw("create file error: ", infoFilePath)
		return false, err
	}
	defer file.Close()

	file.Write(data)

	return true, nil
}
