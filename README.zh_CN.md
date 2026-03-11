# 喜鹊

[English](./README.md) / 中文

deb、appimage、flatpak 转玲珑包工具。

Pica，来自拉丁语的喜鹊，意思是加强精致的生态和适应性。

## 依赖

### 构建依赖

* golang-dlib-dev (>> 1.8.0+)
* golang (>= 2:1.16)

## 安装

### 从源码构建

1. 确认安装所有依赖
2. 构建:

```bash
make build
```

3. 安装:

```bash
sudo make install
```

## 用法

```bash
ll-pica deb init -w work
ll-pica deb convert -w work
ll-pica appimage convert -f demo.AppImage -i io.github.demo -v 1.0.0.0
ll-pica flatpak convert org.kde.kate --build
ll-pica completion bash > ll-pica
```

deb 转换的配置模板、`linglong.yaml` 新结构和依赖处理方式，见 [usage.md](/home/dengbo/WorkDir/project/linglong-pica/usage.md)。

## 获取帮助

有任何使用方面的问题都可以通过以下途径获取帮助。

- [Gitter](https://gitter.im/orgs/linuxdeepin/rooms)
- [IRC 频道](https://webchat.freenode.net/?channels=deepin)
- [论坛](https://bbs.deepin.org)
- [Wiki](https://wiki.deepin.org/)

## 参与贡献

我们鼓励大家报告问题和贡献代码。

- [开发者代码贡献指南][指南]

[BUILD.zh_CN.md](./BUILD.zh_CN.md) 中记录了从源码构建玲珑的方法.

## 许可证

本项目以 [LGPL-3.0-or-later](LICENSE) 许可证发布。

## Credits and references

- [OStree](https://github.com/ostreedev/ostree)

[指南]: https://github.com/linuxdeepin/developer-center/wiki/Contribution-Guidelines-for-Developers
