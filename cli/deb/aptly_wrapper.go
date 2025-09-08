/*
 * SPDX-FileCopyrightText: 2025 UnionTech Software Technology Co., Ltd.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package deb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pkg.deepin.com/linglong/pica/cli/comm"
	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

// AptlyWrapper aptly命令行工具包装器
type AptlyWrapper struct {
	ConfigDir string
}

// NewAptlyWrapper 创建新的aptly包装器
func NewAptlyWrapper() *AptlyWrapper {
	configDir := comm.AptlyCachePath()
	return &AptlyWrapper{
		ConfigDir: configDir,
	}
}

// MirrorInfo aptly镜像信息结构
type MirrorInfo struct {
	Name          string `json:"Name"`
	ArchiveRoot   string `json:"ArchiveRoot"`
	Distribution  string `json:"Distribution"`
	Components    string `json:"Components"`
	Architectures string `json:"Architectures"`
	Filter        string `json:"Filter"`
}

// PackageInfo aptly包信息结构
type PackageInfo struct {
	Key       string `json:"Key"`
	Filename  string `json:"Filename"`
	Size      int64  `json:"Size"`
	Checksums struct {
		SHA256 string `json:"SHA256"`
	} `json:"Checksums"`
	DownloadURL string `json:"DownloadURL"`
}

// CreateMirror 创建aptly镜像
func (a *AptlyWrapper) CreateMirror(name, source, distro, arch, filter string) error {
	// 确保配置目录存在
	if err := os.MkdirAll(a.ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create aptly config dir: %w", err)
	}

	// 创建基本的aptly配置文件
	configPath := filepath.Join(a.ConfigDir, "aptly.conf")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configContent := `{
  "rootDir": "` + a.ConfigDir + `",
  "downloadConcurrency": 4,
  "downloadSpeedLimit": 0,
  "architectures": [],
  "dependencyFollowSuggests": false,
  "dependencyFollowRecommends": false,
  "dependencyFollowAllVariants": false,
  "dependencyFollowSource": false,
  "gpgDisableSign": false,
  "gpgDisableVerify": true,
  "gpgProvider": "gpg",
  "downloadSourcePackages": false,
  "skipLegacyPool": true,
  "ppaDistributorID": "ubuntu",
  "ppaCodename": "",
  "skipContentsPublishing": false,
  "FileSystemPublishEndpoints": {},
  "S3PublishEndpoints": {},
  "SwiftPublishEndpoints": {}
}`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to create aptly config: %w", err)
		}
	}

	args := []string{
		"mirror",
		"create",
		"-ignore-signatures",
		"-architectures=" + arch,
		"-config=" + configPath,
	}

	if filter != "" {
		args = append(args, "-filter="+filter)
	}

	args = append(args, name, source, distro)

	log.Logger.Debugf("Running aptly command: aptly %s", strings.Join(args, " "))

	_, stderr, err := comm.ExecAndWait(60, "aptly", args...)
	if err != nil {
		log.Logger.Errorf("aptly mirror create failed: %s, stderr: %s", err, stderr)
		return fmt.Errorf("aptly mirror create failed: %w", err)
	}

	return nil
}

// UpdateMirror 更新aptly镜像
func (a *AptlyWrapper) UpdateMirror(name string) error {
	configPath := filepath.Join(a.ConfigDir, "aptly.conf")
	args := []string{
		"mirror",
		"update",
		"-ignore-signatures",
		"-config=" + configPath,
		name,
	}

	log.Logger.Debugf("Running aptly command: aptly %s", strings.Join(args, " "))

	_, stderr, err := comm.ExecAndWait(300, "aptly", args...) // 增加超时时间到5分钟
	if err != nil {
		log.Logger.Errorf("aptly mirror update failed: %s, stderr: %s", err, stderr)
		return fmt.Errorf("aptly mirror update failed: %w", err)
	}

	return nil
}

// ListMirrors 列出所有镜像
func (a *AptlyWrapper) ListMirrors() ([]MirrorInfo, error) {
	args := []string{
		"mirror",
		"list",
		"-config=" + filepath.Join(a.ConfigDir, "aptly.conf"),
		"-raw",
	}

	log.Logger.Debugf("Running aptly command: aptly %s", strings.Join(args, " "))

	stdout, stderr, err := comm.ExecAndWait(30, "aptly", args...)
	if err != nil {
		log.Logger.Errorf("aptly mirror list failed: %s, stderr: %s", err, stderr)
		return nil, fmt.Errorf("aptly mirror list failed: %w", err)
	}

	var mirrors []MirrorInfo
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var mirror MirrorInfo
		if err := json.Unmarshal([]byte(line), &mirror); err != nil {
			log.Logger.Warnf("Failed to parse mirror info: %s", err)
			continue
		}
		mirrors = append(mirrors, mirror)
	}

	return mirrors, nil
}

// SearchPackages 搜索包
func (a *AptlyWrapper) SearchPackages(mirrorName, packageName string) ([]PackageInfo, error) {
	configPath := filepath.Join(a.ConfigDir, "aptly.conf")
	args := []string{
		"mirror",
		"search",
		"-config=" + configPath,
		"-format=json",
		mirrorName,
		packageName,
	}

	log.Logger.Debugf("Running aptly command: aptly %s", strings.Join(args, " "))

	stdout, stderr, err := comm.ExecAndWait(30, "aptly", args...)
	if err != nil {
		log.Logger.Errorf("aptly mirror search failed: %s, stderr: %s", err, stderr)
		return nil, fmt.Errorf("aptly mirror search failed: %w", err)
	}

	var packages []PackageInfo
	if strings.TrimSpace(stdout) == "" {
		return packages, nil
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var pkg PackageInfo
		if err := json.Unmarshal([]byte(line), &pkg); err != nil {
			log.Logger.Warnf("Failed to parse package info: %s", err)
			continue
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

// GetPackageURL 获取包的下载URL
func (a *AptlyWrapper) GetPackageURL(mirrorName, packageName string) (string, error) {
	packages, err := a.SearchPackages(mirrorName, packageName)
	if err != nil {
		return "", err
	}

	if len(packages) == 0 {
		return "", fmt.Errorf("package %s not found in mirror %s", packageName, mirrorName)
	}

	// 返回第一个匹配的包的URL
	return packages[0].DownloadURL, nil
}

// GetPackageSources 获取包的源信息
func (a *AptlyWrapper) GetPackageSources(mirrorName, packageName string) ([]comm.Source, error) {
	packages, err := a.SearchPackages(mirrorName, packageName)
	if err != nil {
		return nil, err
	}

	var sources []comm.Source
	for _, pkg := range packages {
		source := comm.Source{
			Kind:   "file",
			Url:    pkg.DownloadURL,
			Digest: pkg.Checksums.SHA256,
		}
		sources = append(sources, source)
	}

	return sources, nil
}

// Cleanup 清理aptly配置
func (a *AptlyWrapper) Cleanup() error {
	if ret, _ := fs.CheckFileExits(a.ConfigDir); ret {
		log.Logger.Debugf("Cleaning up aptly config dir: %s", a.ConfigDir)
		if _, err := fs.RemovePath(a.ConfigDir); err != nil {
			log.Logger.Warnf("Failed to cleanup aptly config: %v", err)
			return err
		}
	}
	return nil
}
