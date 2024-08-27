PBUILDER_PKG = pbuilder-satisfydepends-dummy
PREFIX = usr
BINARY_DIR = bin
BINARY_NAME = ll-pica
APPIMAGE_CONVERT_BINARY_NAME = ll-appimage-convert
GO_PATH = /tmp/go
GO_CACHE = /tmp/go-cache
GoPath := GOPATH=${GO_PATH}
export GOCACHE=${GO_CACHE}
export GO111MODULE=on

GOTEST = go test -v
GOBUILD = go build -mod vendor  -ldflags '-X pkg.deepin.com/linglong/pica/tools/log.disableLogDebug=yes -X pkg.deepin.com/linglong/pica/cli/cobra.disableLogDebug=yes' -v $(GO_BUILD_FLAGS)
GOBUILDDEBUG = go build -mod vendor -v $(GO_BUILD_FLAGS)


all: build

build:
	install -d ${GO_PATH} ${GO_CACHE}
	CGO_ENABLED=0 ${GoPath} ${GOBUILD} -o ${BINARY_DIR}/${BINARY_NAME} ./cmd/${BINARY_NAME}
	CGO_ENABLED=0 ${GoPath} ${GOBUILD} -o ${BINARY_DIR}/${APPIMAGE_CONVERT_BINARY_NAME} ./cmd/${APPIMAGE_CONVERT_BINARY_NAME}

debug:
	${GoPath} ${GOBUILDDEBUG} -o ${BINARY_DIR}/${BINARY_NAME} ./cmd/${BINARY_NAME}
	${GoPath} ${GOBUILDDEBUG} -o ${BINARY_DIR}/${APPIMAGE_CONVERT_BINARY_NAME} ./cmd/${APPIMAGE_CONVERT_BINARY_NAME}

test:
	${GoPath} ${GOTEST} ./tools/...

install:
	install -Dm0755 ${BINARY_DIR}/${BINARY_NAME} ${DESTDIR}/${PREFIX}/${BINARY_DIR}/${BINARY_NAME}
	install -Dm0755 ${BINARY_DIR}/${APPIMAGE_CONVERT_BINARY_NAME} ${DESTDIR}/${PREFIX}/${BINARY_DIR}/${APPIMAGE_CONVERT_BINARY_NAME}

	install -d ${DESTDIR}/${PREFIX}/share/linglong/builder/helper/
	install -Dm0755 misc/libexec/linglong/builder/helper/install_dep ${DESTDIR}/${PREFIX}/libexec/linglong/builder/helper/install_dep
clean:
	rm -rf ${BINARY_DIR}
	rm -rf ${APPIMAGE_CONVERT_BINARY_NAME}
	rm -rf ${GO_PATH}
	rm -rf ${GO_CACHE}

.PHONY: ${BINARY_NAME}
.PHONY: ${APPIMAGE_CONVERT_BINARY_NAME}
