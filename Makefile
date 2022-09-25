PBUILDER_PKG = pbuilder-satisfydepends-dummy
PREFIX = /usr
BINARY_NAME = ll-pica
pwd := ${shell pwd}
GoPath := GOPATH=${pwd}:${pwd}/vendor:${GOPATH}
export GOCACHE=${pwd}/vendor/.cache/go-build

GOTEST = go test -v
GOBUILD = go build -ldflags '-X ll-pica/utils/log.disableLogDebug=yes -X main.disableDevelop=yes' -v $(GO_BUILD_FLAGS)
GOBUILDDEBUG = go build -v $(GO_BUILD_FLAGS)

export GO111MODULE=off

all: build

build:
	${GoPath} ${GOBUILD} -o ${BINARY_NAME} ${BINARY_NAME}

debug:
	${GoPath} ${GOBUILDDEBUG} -o ${BINARY_NAME} ${BINARY_NAME}

test:
	${GoPath} ${GOTEST} ./...

install:
	install -Dm0755 ll-pica ${DESTDIR}/${PREFIX}/bin/ll-pica

clean:
	rm -rf ${BINARY_NAME}
	rm -rf ${pwd}/vendor/.cache

.PHONY: ${BINARY_NAME}
