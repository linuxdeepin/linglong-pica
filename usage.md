# ll-pica使用
## 安装
### 手动编译安装
- 配置go环境
参考 [配置go开发环境](https://blog.csdn.net/qq_41648043/article/details/117782776)
或者：安装deb配置go环境。
```
sudo apt update
sudo apt install golang-go golang-dlib-dev
```
- 下载代码
源码[linglong-pica](https://gitlabwh.uniontech.com/wuhan/v23/linglong/linglong-pica)
- 安装release版本（未开启开发者模式，日志调试模式关闭。）
```
git clone https://gitlabwh.uniontech.com/wuhan/v23/linglong/linglong-pica.git
git checkout develop/snipe
cd linglong-pica
make
sudo make install
```

- 安装debug版本（开启开发者模式，日志调试模式开启。）
```
git clone https://gitlabwh.uniontech.com/wuhan/v23/linglong/linglong-pica.git
git checkout develop/snipe
cd linglong-pica
make debug
sudo make install
```

- 手动安装使用依赖包，当deb包安装时无需手动下载
```
sudo apt update
sudo apt install rsync linglong-builder
```

## 工具说明
本工具目前提供deb包转换为玲珑包的能力。本工具需要提供对于被转换的目标的描述文件，通过描述文件
可以配置转换所需的环境和资源，同时描述文件可以通过定制，干预转换过程。

### 工具安装

本工具当前主要在deepin/UOS系统上适配，deepin/UOS系统可以通过添加如下仓库

- 仓库（社区版本）
`deb https://community-packages.deepin.com/beige/ beige main commercial community`

- 仓库（专业版本）
  敬请期待。

- 下载安装
```
sudo apt update
sudo apt install linglong-pica
```

## 工具使用
### 参数介绍
ll-pica是本工具的命令行工具，主要包含转换环境的初始化、转包、上传玲珑包等功能。

查看ll-pica帮助信息：

`ll-pica --help`

ll-pica帮助信息显示如下：

```
Convert the deb to linglong. For example:
Simple:
        ll-pica init -c runtime.yaml -w work-dir
        ll-pica convert -c app.yaml -w work-dir
        ll-pica push -i appid -w work-dir
        ll-pica help

Usage:
  ll-pica [flags]
  ll-pica [command]

Available Commands:
  convert     Convert deb to linglong
  help        Help about any command
  init        init sdk runtime env
  push        push app to repo

Flags:
  -h, --help      help for ll-pica
  -v, --verbose   verbose output

Use "ll-pica [command] --help" for more information about a command.
```
ll-pica包含init、convert、push等命令参数
- init runtime环境初始化。
- convert 转包操作。
- push上传玲珑包操作。
### 环境初始化
通过使用ll-pica的init命令，对转换所需的环境初进行始化。

通过`ll-pica init --help`命令的查找帮助信息：

ll-pica init 帮助信息显示如下：
```
init sdk runtime env with iso and runtime .

Usage:
  ll-pica init [flags]

Flags:
  -c, --config string    config
  -h, --help             help for init
  -w, --workdir string   work directory

Global Flags:
  -v, --verbose   verbose output
```
运行ll-pica init命令初始化runtime环境：
`ll-pica init -c runtime.yaml  -w  workdir`

#### 参数说明
config参数，-c, --config 指定配置环境的配置文件。

配置文件模板如下：

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

模板字段说明
- base字段为必须配置，配置对应的iso下载地址与玲珑runtime仓库地址。
- repo字段为必须配置，添加对应的仓库地址，支持多仓库添加。
- package字段为备选配置，初始化环境添加额外包安装。
- command字段为备选配置，需要对runtime环境进行的命令参数。

工作目录参数，-w, --workdir 指定工作目录

### 转包
通过使用 `ll-pica convert `命令进行转包。

ll-pica convert帮助信息显示如下：

```
Convert the deb to linglong For example:
Convert:
        ll-pica init
        ll-pica convert  --config config.yaml --workdir=/mnt/workdir

Usage:
  ll-pica convert [flags]

Flags:
  -c, --config string    config
  -h, --help             help for convert
  -w, --workdir string   work directory

Global Flags:
  -v, --verbose   verbose output
```

执行`ll-pica convert  -c config.yaml -w /mnt/workdir`命令进行转包：
#### 参数说明
config参数，-c, --config 指定转包配置文件。

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
      hash:0ffd3af5467acf71320bd2e2267e989c5c4c41abc9a512242d4c37cd42777af5
    - type: localfs
      ref: /tmp/deepin-calculator2_5.7.20-1_amd64.deb
      name: deepin-calculator2
      hash: d38913817d727bca31c1295bae87c02ab129a57172561e3ec8caee6687e03796
  add-package:
    - libicu63
chroot:
  pre-command: |
    uname -m
  post-command: |
    uname -a
```

- info字段为必选，指定软件包信息。

  appid字段为必选，该字段指定正确，才能保证数据获取正确。

  name字段为必选，指定软件包名称，需要与deb包名称一致。

  version字段为推荐使用，用于指定生成的linglong包的版本号，若未指定时，通过deb包版本号进行提取（若deb包版本号，规则与玲珑包规则不一致，则会出现版本号截断情况，因此推荐指定该字段）。

  description字段为可选使用，用于描述玲珑包的软件信息，若被指定时，则使用指定的描述内容。若未指定时，通过转换的软件包提取描述信息。

  kind字段为指定软件包类型，qt、python、qt等，以获取对应转包模板，保证转包正确性。

- file字段为必选，指定软件包信息。

  type字段为必选，deb包获取方式。可选，repo和localfs，其中repo标识表示deb包通过网络获取，localfs标识表示deb包通过本地文件系统获取。。

  ref字段为必选，deb获取地址

  name字段为必选，deb名称。

  hash字段为必选，文件的hash校验值，采用sha256算法计算。

  add-package字段为可选，需要额外安装的deb包。

- chroot字段为可选，转包环境中需要额外执行的命令。

  pre-command字段，在进行转包直接执行，用于修改和控制转换的环境。

  post-command字段，在应用包解压后执行的命令，用于正对应用数据进行修改和处理。


工作目录参数， -w, --workdir 指定工作目录。

### 上传玲珑包
通过使用`ll-pica push`命令用于玲珑包上传仓库。

查看ll-pica push帮助信息：

`ll-pica push --help`
ll-pica convert帮助信息显示如下：

```
Push app to repo that used ll-builder push For example:
push:
        ll-pica push -u deepin -p deepin -i appid -w workdir

Usage:
  ll-pica push [flags]

Flags:
  -i, --appid string       app id
  -c, --channel string     app channel (default "linglong")
  -h, --help               help for push
  -p, --passwords string   passwords
  -r, --repo string        repo url
  -n, --reponame string    repo name
  -u, --user string    username
  -w, --workdir string     work directory

Global Flags:
  -v, --verbose   verbose output
```

运行ll-pica push命令如下：
`ll-pica push -u deepin -p deepin -i org.deepin.calculator -w work-dir`

#### 参数说明

-i, --appid 指定app id 名。
-c, --channel 指定channel,默认为linglong。
-u, --user 指定上传账号。
-p, --passwords 指定上传账号密码。
-r, --repo 指定上传仓库url。
-n,--reponame 指定上传仓库名。
-w,--workdir 指定工作目录。