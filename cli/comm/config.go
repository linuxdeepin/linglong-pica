/*
 * SPDX-FileCopyrightText: 2024 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package comm

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"

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

// 定义 states.json 的结构体
type States struct {
	Config struct {
		DefaultRepo string `json:"defaultRepo"`
		Repos       []struct {
			Name     string `json:"name"`
			Priority int    `json:"priority"`
			Url      string `json:"url"`
		} `json:"repos"`
		Version int `json:"version"`
	} `json:"config"`
	Layers []struct {
		Commit string `json:"commit"`
		Info   struct {
			Arch          []string               `json:"arch"`
			Base          string                 `json:"base"`
			Channel       string                 `json:"channel"`
			Command       []string               `json:"command,omitempty"`
			Description   string                 `json:"description"`
			Id            string                 `json:"id"`
			Kind          string                 `json:"kind"`
			Module        string                 `json:"module"`
			Name          string                 `json:"name"`
			Runtime       string                 `json:"runtime"`
			SchemaVersion string                 `json:"schema_version"`
			Size          int64                  `json:"size"`
			Version       string                 `json:"version"`
			Permissions   map[string]interface{} `json:"permissions,omitempty"`
		} `json:"info"`
		Repo string `json:"repo"`
	} `json:"layers"`
	LlVersion string        `json:"ll-version"`
	Merged    []interface{} `json:"merged"`
	Version   string        `json:"version"`
}

// base runtime 默认优先级由
func NewConfig() *Config {
	return &Config{
		Id:            "org.deepin.runtime.dtk",
		BaseId:        "org.deepin.base",
		Version:       "25.2.0",
		BaseVersion:   "25.2.0",
		Source:        "https://community-packages.deepin.com/beige/",
		DistroVersion: "beige",
		Arch:          runtime.GOARCH,
	}
}

// 读取 pica 配置文件
func (c *Config) ReadConfigJson() bool {
	log.Logger.Infof("load %s", PicaConfigJsonPath())
	picaConfigFd, err := os.ReadFile(PicaConfigJsonPath())
	if err != nil {
		log.Logger.Errorf("load  %s error: %v", PicaConfigJsonPath(), err)
	} else {
		err = json.Unmarshal([]byte(picaConfigFd), &c)
		if err != nil {
			log.Logger.Errorf("unmarshal error: %s", err)
		}

		if strings.HasPrefix(c.BaseVersion, "20") {
			c.BaseId = "org.deepin.foundation"
			c.Id = "org.deepin.Runtime"
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

func GetBaseRuntimeCommit(id, versionPrefix string) string {
	data, err := os.ReadFile(StatesJson)
	if err != nil {
		log.Logger.Errorf("not found states.json in %s: %v", StatesJson, err)
		return ""
	}
	var states States
	if err := json.Unmarshal(data, &states); err != nil {
		log.Logger.Errorf("unmarshal error: %s", err)
		return ""
	}
	for _, layer := range states.Layers {
		if layer.Info.Id == id {
			if strings.Join(strings.Split(layer.Info.Version, ".")[:3], ".") == versionPrefix {
				return layer.Commit
			}
		}
	}
	return "" // 没找到返回空字符串
}
