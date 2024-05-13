package config

import (
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
	"pkg.deepin.com/linglong/pica/cli/deb"
	"pkg.deepin.com/linglong/pica/tools/fs"
	"pkg.deepin.com/linglong/pica/tools/log"
)

const PackageConfigTMPL = `runtime:
  version: {{.Runtime.Version}}
  base_version: {{.Runtime.BaseVersion}}
  source: {{.Runtime.Source}}
  distro_version: {{.Runtime.DistroVersion}}
  arch: {{.Runtime.Arch}}
file:
  deb:
  {{- range $deb := .File.Deb }}
  {{  printf "  - type: %s" $deb.Type}}
  {{  printf "    id: %s" $deb.Id}}
  {{  printf "    name: %s" $deb.Name}}
{{- if ne $deb.Ref ""}}
  {{  printf "    ref: %s" $deb.Ref}}
{{- end}}
{{- if ne $deb.Hash ""}}
  {{  printf "    hash: %s" $deb.Hash}}
{{- end}}
{{end}}
`

type PackConfig struct {
	Runtime struct {
		Config `yaml:",inline"`
	} `yaml:"runtime"`
	File struct {
		Deb []deb.Deb `yaml:"deb"`
	} `yaml:"file"`
}

func NewPackConfig() *PackConfig {
	return &PackConfig{
		Runtime: struct {
			Config `yaml:",inline"`
		}{
			Config: *NewConfig(),
		}, File: struct {
			Deb []deb.Deb `yaml:"deb"`
		}{
			Deb: []deb.Deb{
				{
					Type: "repo",
					Id:   "com.baidu.baidunetdisk",
					Name: "baidunetdisk",
					Ref:  "https://com-store-packages.uniontech.com/appstorev23/pool/appstore/c/com.baidu.baidunetdisk/com.baidu.baidunetdisk_4.17.7_amd64.deb",
					Hash: "db7ad7b6af9746f968328737b0893c96b0755958916c34d8b1f9241047505400",
				},
				{
					Type: "repo",
					Id:   "com.baidu.baidunetdisk",
					Name: "baidunetdisk",
					Ref:  "/tmp/com.baidu.baidunetdisk_4.17.7_amd64.deb",
				},
				{
					Type: "repo",
					Id:   "com.baidu.baidunetdisk",
					Name: "baidunetdisk",
				},
				{
					Type: "local",
					Id:   "com.baidu.baidunetdisk",
					Name: "baidunetdisk",
					Ref:  "/tmp/com.baidu.baidunetdisk_4.17.7_amd64.deb",
				},
			},
		},
	}
}

func (p *PackConfig) ReadPackConfigYaml(path string) bool {
	log.Logger.Infof("load %s", path)
	packConfigFd, err := os.ReadFile(path)
	if err != nil {
		log.Logger.Errorf("load %s error: %v", path, err)
	} else {
		if err = yaml.Unmarshal(packConfigFd, p); err != nil {

			log.Logger.Errorf("unmarshal %s error: %v", path, err)
		}
		return true
	}
	return false
}

func (p *PackConfig) CreatePackConfigYaml(path string) bool {
	tpl, err := template.New("package").Parse(PackageConfigTMPL)

	if err != nil {
		log.Logger.Warnf("parse template failed: %v", err)
		return false
	}

	// 创建配置 package.yaml 文件
	if ret, _ := fs.CheckFileExits(path); !ret {
		// create save file
		log.Logger.Infof("create save file: %s", path)
		saveFd, ret := os.Create(path)
		if ret != nil {
			log.Logger.Fatalf("save to %s failed!", path)
			return false
		}
		defer saveFd.Close()

		// render template
		log.Logger.Debug("render template: ", p)
		tpl.Execute(saveFd, p)
	} else {
		log.Logger.Infof("%s is exited", path)
		return false
	}

	return true
}
