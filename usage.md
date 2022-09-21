# ll-pica使用
## 安装
### 手动编译安装
- 配置go环境
参考 [配置go开发环境](https://blog.csdn.net/qq_41648043/article/details/117782776)
- 下载代码
源码[linglong-pica](https://gitlabwh.uniontech.com/wuhan/v23/linglong/linglong-pica)
```
git clone https://gitlabwh.uniontech.com/wuhan/v23/linglong/linglong-pica.git
git checkout develop/snipe
cd linglong-pica
make -j8
make install
```
- 手动安装使用依赖包，当deb包安装时无需手动下载
```
sudo apt update
sudo apt install rsync linglong-builder
```
### 下载deb安装
- 添加下面仓库
`deb https://xxxxxxxxxxxx eagle main`
- 下载安装
```
sudo apt update
sudo apt install linglong-pica
```
## 命令行工具使用
### ll-pica简介
ll-pica是一个deb转玲珑包命令行工具，包含环境初始化、转包、上传玲珑包等功能。

查看ll-pica帮助信息：

`ll-pica --help`

ll-pica帮助信息显示如下：
```
Convert the deb to uab. For example:
Simple:
        ll-pica init 
        ll-pica convert -d abc.deb --config config.yaml -w /mnt/workdir
        ll-pica help

Usage:
  ll-pica [flags]
  ll-pica [command]

Available Commands:
  convert     Convert deb to uab
  help        Help about any command
  init        init sdk runtime env
  push        push uab to repo

Flags:
  -h, --help      help for ll-pica
      --verbose   verbose output

Use "ll-pica [command] --help" for more information about a command.
```
ll-pica包含init、convert、push等命令参数
- init runtime环境初始化。
- convert 转包操作。
- push上传玲珑包操作。
### runtime环境初始化
ll-pica init 命令用于runtime环境初始化。
查看ll-pica init 命令的帮助信息：
`ll-pica init --help`
ll-pica init 帮助信息显示如下：
```
init sdk runtime env with iso and runtime .

Usage:
  ll-pica init [flags]

Flags:
  -c, --config string    config
  -h, --help             help for init
  -k, --keep-cached      keep cached (default true)
  -w, --workdir string   work directory

Global Flags:
      --verbose   verbose output
```
运行ll-pica init命令初始化runtime环境：
`ll-pica init -c runtime.yaml  -w  workdir`

-c, --config 指定runtime配置文件
runtime配置文件模板如下：

``` 
---
sdk:
  base:
    -
      type: iso
      ref: https://cdimage.uniontech.com/iso-v20/uniontechos-desktop-20-professional-1050-amd64.iso
      hash: 18b7ccaa77abf96eaa5eee340838d9ccead006bfb9feba3fd3da30d58e292a17
    - 
      type: ostree
      ref: linglong/org.deepin.Runtime/20.5.0/x86_64/runtime
      hash:
      remote: https://repo.linglong.space/repo
  extra:
    repo:
      - "deb [trusted=yes] http://pools.uniontech.com/desktop-professional/ eagle main contrib non-free"
    package:
      - libicu63
    command: |
      apt update
      # disable dde triggers
      [[ -f /var/lib/dpkg/triggers/File ]] && ( sed -i 's|/opt/apps\s\S*$||' /var/lib/dpkg/triggers/File )
```
- base必须配置，配置对应的iso下载地址与玲珑runtime仓库地址。
- repo必须配置，添加对应的仓库地址，支持多仓库添加。
- package备选配置，初始化环境添加额外包安装。
- command备选配置，需要对runtime环境进行的命令参数。

-w, --workdir 指定工作目录
-k, --keep-cached 读取缓存初始化环境

### deb转玲珑包
ll-pica convert命令用于deb转玲珑包。
查看ll-pica convert帮助信息：
`ll-pica convert --help`
ll-pica convert帮助信息显示如下：

```
Convert the deb to uab For example:
Convert:
        ll-pica init
        ll-pica convert --deb abc.deb --config config.yaml --workdir=/mnt/workdir

Usage:
  ll-pica convert [flags]

Flags:
  -c, --config string     config
  -d, --deb-file string   deb file
  -h, --help              help for convert
  -w, --workdir string    work directory

Global Flags:
      --verbose   verbose output
```
运行ll-pica convert命令如下：
`ll-pica convert -d abc.deb -c config.yaml -w /mnt/workdir`

-d, --deb-file 指定本地deb包进行转包。注意：当指定本地包时，优先转本地deb包，配置文件中无需指定deb包下载信息。

-c, --config 指定deb转包配置文件。
配置文件模板如下：

```
---
info:
  appid: org.deepin.calculator
  name: deepin-calculator
  version: 5.7.20.1
  description: calculator for deepin os\n
  kind: qt
file:
  deb:
    - type: repo
      ref: http://pools.uniontech.com/desktop-professional/pool/main/d/deepin-calculator/deepin-calculator_5.7.20-1_amd64.deb
      name: deepin-calculator
      hash: 0ffd3af5467acf71320bd2e2267e989c5c4c41abc9a512242d4c37cd42777af5
  add-package:
    - libicu63
chroot:
  pre-command: |
    uname -m
  post-command: |
    uname -a
```
- info: 指定软件包信息。
appid：必须指定正确，才能保证数据获取正确。
name:  指定软件包名称。
version:  可根据需求指定，当指定时，以指定版本为准，未指定时，通过deb包获取版本号。
description: 可根据需求指定，当指定时，以指定描述为准，未指定时，通过deb包获取描述信息。
kind:  指定软件包类型，qt、python、qt等，以获取对应转包模板，保证转包正确性。
- file: 指定软件包信息，当转本地deb包时，无需填写此参数。
type: deb包获取方式。
ref: deb获取地址。
name: deb名称。
hash: hash校验值。
add-package: 需要额外安装的deb包。
- chroot: 转包环境中需要额外执行的命令。
pre-command: 安装应用前执行的命令。
post-command: 安装应用后执行的命令。

-w, --workdir 指定工作目录。

### 上传玲珑包
ll-pica push命令用于玲珑包上传仓库。
查看ll-pica push帮助信息：
`ll-pica push --help`
ll-pica convert帮助信息显示如下：

```
Push uab to repo that used ll-builder push For example:
push:
        ll-pica push -u deepin -p deepin -d org.deepin.calculator_x86-64.uab

Usage:
  ll-pica push [flags]

Flags:
  -c, --channel string     bundle channel (default "linglong")
  -h, --help               help for push
  -k, --keyfile string     auth key file
  -p, --passwords string   passwords
  -r, --repo string        bundle repo url
  -d, --uab string         bundle path
  -u, --username string    username

Global Flags:
      --verbose   verbose output
```
运行ll-pica push命令如下：
`ll-pica push -u deepin -p deepin -d org.deepin.calculator_x86-64.uab`

-c, --channel 指定channel,默认为linglong。
-k, --keyfile 指定授权配置文件，当未指定时，需要使用-u -p参数指定。
-u, --username 指定上传账号。
-p, --passwords 指定上传账号密码。
-r, --repo 指定上传仓库url。
-d, --uab 指定需要上传的uab包。