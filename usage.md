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

```bash
sudo apt update
sudo apt install linglong-builder
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

ll-pica是本工具的命令行工具，主要包含转换环境的初始化、转包等功能。

查看ll-pica帮助信息：

`ll-pica --help`

ll-pica帮助信息显示如下：

```bash
Convert the deb to uab. For example:
Simple:
        ll-pica init -c package -w work-dir
        ll-pica convert -c package.yaml -w work-dir
        ll-pica help

Usage:
  ll-pica [command]

Available Commands:
  convert     Convert deb to uab
  help        Help about any command
  init        init config template

Flags:
  -h, --help      help for ll-pica
  -V, --verbose   verbose output
  -v, --version   version for ll-pica

Use "ll-pica [command] --help" for more information about a command.
```

ll-pica包含init、convert 命令参数

- init 初始化模板。
- convert 转包操作。

### 环境初始化

通过使用ll-pica的init命令，对转换所需的环境初进行始化。

通过 `ll-pica init --help`命令的查找帮助信息：

ll-pica init 帮助信息显示如下：

```bash
init config template

Usage:
  ll-pica init [flags]

Flags:
  -a, --arch string      runtime arch
  -c, --config string    config file
      --dv string        distribution Version
  -h, --help             help for init
      --pi string        package id
      --pn string        package name
  -s, --source string    runtime source
  -t, --type string      get type
  -v, --version string   runtime version
  -w, --workdir string   work directory

Global Flags:
  -V, --verbose   verbose output
```

运行ll-pica init命令初始化runtime环境：
`ll-pica init -c package -w workdir` 或 `ll-pica init`

#### 参数说明

-c, --config 指定配置的模板类型，目前只有 package.yaml，且为默认参数，可以不进行指定参数，可以直接指定 deb 包作为参数。

-w, --workdir 工具的工作目录，下载 deb 包，解压文件，生成 linglong.yaml 都会在该工作目录下，可以不指定参数，默认路径为 `~/.cache/linglong-pica`。

-a, --arch 设置需要转换的架构，可选（amd64, arm64)。

--dv, 发行代号，(beige、eagle/1063、eagle/1070等)。

--pi，对应玲珑包唯一识别名称。

--pn, 对应 apt 安装时的名称。

-s, --source，apt 使用的源，如 http://pools.uniontech.com/desktop-professional/ 。

-t, --type，获取方式 repo，从 apt 软件仓库获取，local 本地获取，需要指定本地路径。

-v, --version，runtime 和 base 环境版本，写三位版本号，如 20.0.0，留空一位便于模糊匹配，runtime 和 base 版本更新后不用重新打包。

配置文件模板如下：

```bash
runtime:
  version: 20.0.0
  source: http://pools.uniontech.com/desktop-professional
  distro_version: eagle/1063
  arch: amd64
file:
  deb:
    - type: repo
      id: com.baidu.baidunetdisk
      name: com.baidu.baidunetdisk
      ref: https://com-store-packages.uniontech.com/appstorev23/pool/appstore/c/com.baidu.baidunetdisk/com.baidu.baidunetdisk_4.17.7_amd64.deb

    - type: repo
      id: com.qq.wemeet
      name: wemeet
      ref: https://com-store-packages.uniontech.com/appstorev23/pool/appstore/c/com.qq.wemeet/com.qq.wemeet_3.19.0.401_amd64.deb

    - type: local
      id: com.baidu.baidunetdisk
      name: baidunetdisk
      ref: /tmp/com.baidu.baidunetdisk_4.17.7_amd64.deb

```

模板字段说明

- runtime 字段为必须配置，需要运行玲珑应用的基础环境，指定包名和版本。是读取 ~/.pica/config.json 文件的默认配置，可修改。

  - version 字段必须配置，runtime 环境版本号。
  - source 字段必须配置，ll-pica 通过 aptly 获取软件依赖，需要指定一下软件源。
  - distro_version 必选配置，发行版代号。
  - arch 字段必须配置，软件的架构，这样与宿主机架构分离，可选择转换其他架构的软件。
- file 字段为必须配置，需要转换的包文件类型。

  - deb 字段为必须配置，表示 deb 包类型的包
    - type 字段必须配置，获取包的方式，repo 指定 url 下载，local 指定本地路径。
    - id 字段必须配置，对应玲珑包唯一识别名称。
    - name 字段为必须配置，软件包名称, 使用 apt 安装时候用的包名。
    - ref 字段被可选配置，如果指定了 type 为 repo ，就使用 url 地址，并且 ref 留空，使用 apt 自动查询源里可用的，如果指定 type 为 local, 就指定本地绝对路径。
    - hash 字段备选配置，如果为空不进行 hash 验证，否则进行验证。

### 转包

通过使用 `ll-pica convert `命令进行转包。

ll-pica convert帮助信息显示如下：

```bash
Convert deb to uab

Usage:
  ll-pica convert [flags]

Flags:
  -b, --build            build linglong
  -c, --config string    config file
  -h, --help             help for convert
      --pi string        package id
      --pn string        package name
  -t, --type string      get app type (default "local")
  -w, --workdir string   work directory

Global Flags:
  -V, --verbose   verbose output
```

执行 `ll-pica convert  -c config.yaml -w /mnt/workdir`命令进行转包：

#### 参数说明

config，-c, --config 指需要转包的配置文件，默认参数为package.yaml，也可以传入一个 xx.deb 文件。

workdir，-w, --workdir 工具的工作目录，下载 deb 包，解压文件，生成 linglong.yaml 都会在该工作目录下，可以不指定参数，默认路径为 `~/.cache/linglong-pica`。

--pi，对应玲珑包唯一识别名称。

--pn, 对应 apt 安装时的名称。

type, -t, --type 获取方式， repo 从 apt 仓库中获取，local 表示本地需要指定路径。

build，-b, --build 指需要进行玲珑包构建，默认参数为 false，如果为 true 生成 linglong.yaml 文件并进行构建导出 layer 文件。

### 具体使用

#### 通过包名转换

首先需要知道包名，以百度网盘为例子，如果不知道包名可以先在应用商店把应用安装了。

然后把应用的菜单栏找到百度网盘的图标右键，发送到桌面。在桌面的 DESKTOP  文件右键选择一个编辑器打开，找到  Exec 这行，应用商店会把应用的 APPID 拼接在 /opt/apps 后面，这个是应用的唯一标识。同时在 apt 中也是使用这个作为应用的包名。此时我们知道包名可以进行后面的操作。

`ll-pica init -w work` 初始化配置，生成默认配置。-w 指定工作目录，在运行命令的当前目录下生成 work 工作目录，不指定则直接使用 `~/.cache/linglong-pica` 。

```bash
ll-pica init -w w --pi com.baidu.baidunetdisk --pn com.baidu.baidunetdisk -t repo
```

init 表示初始化模板。

-w 表示工作目录，后面是目录名，直接在当前路径生成 w 目录，如果不指定这个参数，默认工作目录 ~/.linglong-pica。

--pi 表示玲珑应用的包名。

--pn 表示 apt 使用的包名。

-t 表示获取方式，repo 从 apt 仓库中拉取。

执行完命令会生成 package.yaml 文件，这个用来存放后面需要转换的 deb 包，会生成如果的模板。runtime 字段的相关配置，是读取 ~/.pica/config.json 获取的，如果需要修改默认生成的配置可以修改该文件。另外的如果需要批量转换，可以将 多个包写在 package.yaml 文件。如下：

```bash
runtime:
  version: 20.0.0
  source: http://pools.uniontech.com/desktop-professional
  distro_version: eagle/1063
  arch: amd64
file:
  deb:
    - type: repo
      id: com.baidu.baidunetdisk
      name: com.baidu.baidunetdisk
  
    - type: repo
      id: com.qq.wemeet
      name: com.qq.wemeet
```

在工作目录下会生成 package.yaml，添加需要转换的包，批量转换。

`ll-pica convert -w work` 对工作目录的包进行转换生成 linglong.yaml 文件，加上 -b 参数进行构建。

deb 字段为必须配置，表示 deb 包类型的包

type 字段必须配置，获取包的方式，repo 指定 url 下载，local 指定本地路径。

id 字段必须配置，对应玲珑包唯一识别名称。

name 字段为必须配置，软件包名称, 使用 apt 安装时候用的包名。

**进行转换**

```
ll-pica convert -w w
```

-w 工作目录，在当前目录下的 w 目录，如果不指定这个参数，默认工作目录 ~/.linglong-pica。

转换后生成的目录如下。linglong.yaml 文件是 玲珑打包需要的描述文件。sources 目录下存放下载的文件，如果没法在 linglong.yaml 文件添加 deb 包的 url 链接，可以手动把 deb 包放到 sources 目录下，不过这样不能保证可重复打包。

```bash
.
├── package
│   └── com.baidu.baidunetdisk
│       └── amd64
│           ├── linglong
│           │   └── sources
│           └── linglong.yaml
└── package.yaml
```

前面 init 和 convert 可以直接用一条命令

```bash
ll-pica convert -w w --pi com.baidu.baidunetdisk --pn com.baidu.baidunetdisk
```

#### 通过已有 deb 包转换

假设手上了 deb 包，可能无法用 url 下载，比如我有一份 微信旧版的 deb包，直接用转包。

```bash
ll-pica convert -c com.qq.weixin.deepin.deb -w w
```

#### linglong.yaml

通过 ll-pica convert 命令转换之后生成 linglong.yaml 文件。

```bash
version: "1"

package:
  id: com.baidu.baidunetdisk
  name: com.baidu.baidunetdisk
  version: 4.17.7.0
  kind: app
  description: |
    com.baidu.baidunetdisk

base: org.deepin.foundation/20.0.0
runtime: org.deepin.Runtime/20.0.0

command:
  - "/opt/apps/com.baidu.baidunetdisk/files/baidunetdisk/baidunetdisk"

sources:
  - kind: file
    url: https://com-store-packages.uniontech.com/appstorev23/pool/appstore/c/com.baidu.baidunetdisk/com.baidu.baidunetdisk_4.17.7_amd64.deb
    digest: db7ad7b6af9746f968328737b0893c96b0755958916c34d8b1f9241047505400
  
  - kind: file
    url: http://pools.uniontech.com/desktop-professional/pool/main/b/bzip2/libbz2-1.0_1.0.6.2-deepin2_amd64.deb
    digest: 33edbf929da4746ff995f67350c8ca8b9c872908d6de61493b3d26021556d843
build: |
  #>>> auto generate by ll-pica begin
  SOURCES="linglong/sources"
  # remove extra desktop file
  find $SOURCES/app -name "*.desktop"|grep "uos"|xargs -I {} rm {}
  export TRIPLET="x86_64-linux-gnu"
  export PATH=$PATH:/usr/libexec/linglong/builder/helper
  install_dep $SOURCES $PREFIX
  # modify desktop, Exec and Icon should not contanin absolut paths
  desktopPath=`find $SOURCES/app -name "*.desktop" | grep entries`
  sed -i '/Exec*/c\Exec=start.sh --no-sandbox %U' $desktopPath
  sed -i '/Icon*/c\Icon=baidunetdisk' $desktopPath
  # use a script as program
  echo "#!/usr/bin/env bash" > start.sh
  echo "export LD_LIBRARY_PATH=/opt/apps/com.baidu.baidunetdisk/files/lib/x86_64-linux-gnu:/runtime/lib:/runtime/lib/x86_64-linux-gnu:/usr/lib:/usr/lib/x86_64-linux-gnu" >> start.sh
  echo "cd $PREFIX/baidunetdisk && ./baidunetdisk \$@" >> start.sh
  install -d $PREFIX/share
  install -d $PREFIX/bin
  install -d $PREFIX/lib
  install -m 0755 start.sh $PREFIX/bin
  # move files
  cp -r $SOURCES/app/opt/apps/com.baidu.baidunetdisk/entries/* $PREFIX/share
  cp -r $SOURCES/app/opt/apps/com.baidu.baidunetdisk/files/* $PREFIX
  #>>> auto generate by ll-pica end
```

##### VERSION

linglong.yaml 文件版本

##### package

主要描述了玲珑软件包的元数据。其中玲珑定义了三种项目类型，lib，runtime，app。

 lib ：库文件类型，通常作为构建依赖供runtime类型和app类型引入。

 runtime ：运行时类型，该类型主要提供一组通用的库文件，它其实是lib的集合。

 app ：应用类型，构建出的软件包能够安装和运行。

version：应用版本号，需要指定四位。

##### base

构建使用的基础环境，目前仓库里的base为org.deepin.foundation，主要提供一些常见的编译器和构建系统。如gcc/g++,  clang , cmake等等。

##### runtime

运行时环境,  目前仓库里的runtime为org.deepin.Runtime, 包含qt和dtk的动态库，如果有需要可以制作自己的 runtime。

##### source

源码地址，支持archive、git、local四种类型源码形式，file 直接下载文件。可以是个数组，转包工具会把所有依赖生成列表。linglong-pica 包提供一个，/usr/libexec/linglong/builder/helper/install_dep 脚本，会检测构建时候的 runtime 依赖，对不存在的库，进行安装。

##### build

构建命令，描述构建流程，是在容器内部运行的构建命令。可以使用 ll-builder build --exec bash 来进入容器进行命令调试。

#### 构建应用

进入 linglong.yaml 所在的路径执行命令，构建应用。

```
ll-builder build
```

可以加 -v 参数可以看到更多信息

运行应用

```
ll-builder run
```

可以加 -v 参数可以看到更多信息
