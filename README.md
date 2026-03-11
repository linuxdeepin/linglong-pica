# linglong-pica

deb, appimage and flatpak package to Linglong package tool.

Pica, from the Latin magpie, means to strengthen the exquisite ecology and adaptability.

## Dependencies

### Build dependencies

* golang-dlib-dev (>> 1.8.0+)
* golang (>= 2:1.15)

## Installation

### Build from source code

1. Make sure you have installed all dependencies.

2. Build:

    ```bash
    make build
    ```

3. Install:

    ```bash
    sudo make install
    ```

## Usage

```bash
ll-pica deb init -w work
ll-pica deb convert -w work
ll-pica appimage convert -f demo.AppImage -i io.github.demo -v 1.0.0.0
ll-pica flatpak convert org.kde.kate --build
ll-pica completion bash > ll-pica
```

Deb conversion details, generated `package.yaml`, and the updated `linglong.yaml` layout are documented in [usage.md](/home/dengbo/WorkDir/project/linglong-pica/usage.md).

## Getting help

Any usage issues can ask for help via

* [Telegram group](https://t.me/deepin)
* [Matrix](https://matrix.to/#/#deepin-community:matrix.org)
* [IRC (libera.chat)](https://web.libera.chat/#deepin-community)
* [Forum](https://bbs.deepin.org)
* [WiKi](https://wiki.deepin.org/)

## Getting involved

We encourage you to report issues and contribute changes

* [Contribution guide for developers](https://github.com/linuxdeepin/developer-center/wiki/Contribution-Guidelines-for-Developers-en).

## License

deepin-tool-kit is licensed under [LGPL-3.0-or-later](LICENSE).
