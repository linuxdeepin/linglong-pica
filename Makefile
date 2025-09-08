PREFIX = usr
BINARY_DIR = bin
BINARY_NAME = ll-pica
APPIMAGE_CONVERT_BINARY_NAME = ll-appimage-convert
FLATPAK_CONVERT_BINARY_NAME = ll-pica-flatpak
GOPATH_DIR = gopath
GOPKG_PREFIX = pkg.deepin.com/linglong/pica
GOBUILD = go build -ldflags '-X pkg.deepin.com/linglong/pica/tools/log.disableLogDebug=yes -X pkg.deepin.com/linglong/pica/cli/cobra.disableLogDebug=yes' -v $(GO_BUILD_FLAGS)
GOBUILDDEBUG = go build -v $(GO_BUILD_FLAGS)

export GOPATH=$(shell go env GOPATH)

GOTEST = go test -v


all: build

prepare:
	@mkdir -p ${BINARY_DIR}
	@mkdir -p ${GOPATH_DIR}/src/$(dir ${GOPKG_PREFIX});
	@rm -rf ${GOPATH_DIR}/src/${GOPKG_PREFIX}
	@ln -snf ../../../.. ${GOPATH_DIR}/src/${GOPKG_PREFIX}
	@rm -rf ${GOPATH_DIR}/src/${GOPKG_PREFIX}/vendor


${BINARY_DIR}/${BINARY_NAME}: prepare
	env GOPATH="${CURDIR}/${GOPATH_DIR}:${GOPATH}" ${GOBUILD} -o $@ ./cmd/${BINARY_NAME}

${BINARY_DIR}/${APPIMAGE_CONVERT_BINARY_NAME}: prepare
	env GOPATH="${CURDIR}/${GOPATH_DIR}:${GOPATH}" ${GOBUILD} -o $@ ./cmd/${APPIMAGE_CONVERT_BINARY_NAME}

${BINARY_DIR}/${FLATPAK_CONVERT_BINARY_NAME}: prepare
	cp ./cmd/${FLATPAK_CONVERT_BINARY_NAME}/${FLATPAK_CONVERT_BINARY_NAME} $@

build: ${BINARY_DIR}/${BINARY_NAME} ${BINARY_DIR}/${APPIMAGE_CONVERT_BINARY_NAME} ${BINARY_DIR}/${FLATPAK_CONVERT_BINARY_NAME}

debug:
	env GOPATH="${CURDIR}/${GOPATH_DIR}:${GOPATH}" ${GOBUILDDEBUG} -o ${BINARY_DIR}/${BINARY_NAME} ./cmd/${BINARY_NAME}
	env GOPATH="${CURDIR}/${GOPATH_DIR}:${GOPATH}" ${GOBUILDDEBUG} -o ${BINARY_DIR}/${APPIMAGE_CONVERT_BINARY_NAME} ./cmd/${APPIMAGE_CONVERT_BINARY_NAME}

test:
	env GOPATH="${CURDIR}/${GOPATH_DIR}:${GOPATH}" ${GOTEST} ./tools/...

install:
	install -Dm0755 ${BINARY_DIR}/${BINARY_NAME} ${DESTDIR}/${PREFIX}/${BINARY_DIR}/${BINARY_NAME}
	install -Dm0755 ${BINARY_DIR}/${APPIMAGE_CONVERT_BINARY_NAME} ${DESTDIR}/${PREFIX}/${BINARY_DIR}/${APPIMAGE_CONVERT_BINARY_NAME}
	install -Dm0755 cmd/ll-pica-flatpak/ll-pica-flatpak ${DESTDIR}/${PREFIX}/${BINARY_DIR}/ll-pica-flatpak
	install -Dm0755 cmd/ll-pica-flatpak/ll-pica-flatpak-convert ${DESTDIR}/${PREFIX}/${BINARY_DIR}/ll-pica-flatpak-convert
	install -Dm0755 cmd/ll-pica-flatpak/ll-pica-flatpak-utils ${DESTDIR}/${PREFIX}/${BINARY_DIR}/ll-pica-flatpak-utils
	install -Dm0755 cmd/ll-convert-tool/ll-convert-tool ${DESTDIR}/${PREFIX}/${BINARY_DIR}/ll-convert-tool

	install -d ${DESTDIR}/${PREFIX}/share/linglong/builder/helper/
	install -Dm0755 misc/libexec/linglong/builder/helper/install_dep ${DESTDIR}/${PREFIX}/libexec/linglong/builder/helper/install_dep
clean:
	rm -rf ${BINARY_DIR}
	rm -rf ${GOPATH_DIR}

.PHONY: ${BINARY_NAME}
.PHONY: ${APPIMAGE_CONVERT_BINARY_NAME}
