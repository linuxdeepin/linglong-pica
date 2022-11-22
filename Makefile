PBUILDER_PKG = pbuilder-satisfydepends-dummy
PREFIX = usr
BINARY_DIR = bin
BINARY_NAME = ll-pica
GO_PATH = /tmp/go
GO_CACHE = /tmp/go-cache
pwd := ${shell pwd}
GoPath := GOPATH=${GO_PATH}
export GOCACHE=${GO_CACHE}
export GO111MODULE=on

GOTEST = go test -v
GOBUILD = go build -mod vendor  -ldflags '-X pkg.deepin.com/linglong/pica/cmd/ll-pica/utils/log.disableLogDebug=yes -X main.disableDevelop=yes' -v $(GO_BUILD_FLAGS)
GOBUILDDEBUG = go build -mod vendor -v $(GO_BUILD_FLAGS)


all: build

build:
	install -d ${GO_PATH} ${GO_CACHE}
	${GoPath} ${GOBUILD} -o ${BINARY_DIR}/${BINARY_NAME} ./cmd/${BINARY_NAME}

debug:
	${GoPath} ${GOBUILDDEBUG} -o ${BINARY_DIR}/${BINARY_NAME} ./cmd/${BINARY_NAME}

test:
	${GoPath} ${GOTEST} ./cmd/...

install:
	install -Dm0755  ${BINARY_DIR}/${BINARY_NAME} ${DESTDIR}/${PREFIX}/${BINARY_DIR}/${BINARY_NAME}

clean:
	rm -rf ${BINARY_DIR}
	rm -rf ${GO_PATH}
	rm -rf ${GO_CACHE}

.PHONY: ${BINARY_NAME}
