/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package config

import (
	"encoding/json"
	"os"

	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/tools/log"
)

type Config struct {
	Id            string `yaml:"-" json:"-"`
	BaseId        string `yaml:"-" json:"-"`
	Version       string `yaml:"version" json:"version"`
	BaseVersion   string `yaml:"base_version" json:"base_version"`
	Source        string `yaml:"source" json:"source"`
	DistroVersion string `yaml:"distro_version" json:"distro_version"`
	Arch          string `yaml:"arch" json:"arch"`
}

func NewConfig() *Config {
	return &Config{
		Id:            "org.deepin.Runtime",
		BaseId:        "org.deepin.foundation",
		Version:       "23.0.1",
		BaseVersion:   "23.0.0",
		Source:        "https://community-packages.deepin.com/beige/",
		DistroVersion: "beige",
		Arch:          "amd64",
	}
}

// 读取 pica 配置文件
func (c *Config) ReadConfigJson() bool {
	log.Logger.Infof("load %s", comm.PicaConfigJsonPath())
	picaConfigFd, err := os.ReadFile(comm.PicaConfigJsonPath())
	if err != nil {
		log.Logger.Errorf("load  %s error: %v", comm.PicaConfigJsonPath(), err)
	} else {
		err = json.Unmarshal([]byte(picaConfigFd), &c)
		if err != nil {
			log.Logger.Errorf("unmarshal error: %s", err)
		}
		return true
	}
	return false
}

func (c *Config) SaveOrUpdateConfigJson(path string) bool {
	// 创建 pica 工具配置文件
	log.Logger.Infof("create save file: %s", path)

	jsonBytes, err := json.Marshal(c)
	if err != nil {
		log.Logger.Errorf("JSON marshaling failed: %s", err)
	}

	err = os.WriteFile(path, jsonBytes, 0644)
	if err != nil {
		log.Logger.Fatalf("save to %s failed!", path)
		return false
	}

	return true
}
