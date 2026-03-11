PBUILDER_PKG = pbuilder-satisfydepends-dummy
PREFIX = usr
BINARY_DIR = bin
BINARY_NAME = ll-pica
GOPATH ?= /tmp/go
GOCACHE ?= /tmp/go-cache
VERSION ?= $(shell dpkg-parsechangelog -S Version 2>/dev/null || sed -n '1s/.*(\\(.*\\)).*/\\1/p' debian/changelog 2>/dev/null || echo dev)
export GO111MODULE=on

GOTEST = go test -v
LL_PICA_LDFLAGS = -X pkg.deepin.com/linglong/pica/tools/log.disableLogDebug=yes -X pkg.deepin.com/linglong/pica/cli/cobra.disableLogDebug=yes -X pkg.deepin.com/linglong/pica/cli/version.Version=${VERSION}
GOBUILD = go build -mod vendor -ldflags '$(LL_PICA_LDFLAGS)' -v $(GO_BUILD_FLAGS)
GOBUILDDEBUG = go build -mod vendor -ldflags '-X pkg.deepin.com/linglong/pica/cli/version.Version=${VERSION}' -v $(GO_BUILD_FLAGS)


all: build

build:
	install -d ${GOPATH} ${GOCACHE}
	CGO_ENABLED=0 ${GOBUILD} -o ${BINARY_DIR}/${BINARY_NAME} ./cmd/${BINARY_NAME}

completion: build
	install -d ${BINARY_DIR}/completions
	${BINARY_DIR}/${BINARY_NAME} completion bash > ${BINARY_DIR}/completions/${BINARY_NAME}
	${BINARY_DIR}/${BINARY_NAME} completion zsh > ${BINARY_DIR}/completions/_${BINARY_NAME}
	${BINARY_DIR}/${BINARY_NAME} completion fish > ${BINARY_DIR}/completions/${BINARY_NAME}.fish

debug:
	${GOBUILDDEBUG} -o ${BINARY_DIR}/${BINARY_NAME} ./cmd/${BINARY_NAME}

test:
	${GOTEST} ./tools/...

install:
	install -Dm0755 ${BINARY_DIR}/${BINARY_NAME} ${DESTDIR}/${PREFIX}/${BINARY_DIR}/${BINARY_NAME}
	install -d ${DESTDIR}/${PREFIX}/share/bash-completion/completions
	install -d ${DESTDIR}/${PREFIX}/share/zsh/vendor-completions
	install -d ${DESTDIR}/${PREFIX}/share/fish/vendor_completions.d
	${BINARY_DIR}/${BINARY_NAME} completion bash > ${DESTDIR}/${PREFIX}/share/bash-completion/completions/${BINARY_NAME}
	${BINARY_DIR}/${BINARY_NAME} completion zsh > ${DESTDIR}/${PREFIX}/share/zsh/vendor-completions/_${BINARY_NAME}
	${BINARY_DIR}/${BINARY_NAME} completion fish > ${DESTDIR}/${PREFIX}/share/fish/vendor_completions.d/${BINARY_NAME}.fish

	install -d ${DESTDIR}/${PREFIX}/share/linglong/builder/helper/
	install -Dm0755 misc/libexec/linglong/builder/helper/install_dep ${DESTDIR}/${PREFIX}/libexec/linglong/builder/helper/install_dep
clean:
	rm -rf ${BINARY_DIR}
	rm -rf ${GOCACHE}
	rm -rf ${GOPATH}

.PHONY: ${BINARY_NAME}
.PHONY: completion
