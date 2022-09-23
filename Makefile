PREFIX = /usr
BINARY_NAME = ll-pica
pwd := ${shell pwd}
GoPath := GOPATH=${pwd}:${pwd}/vendor:${GOPATH}

GOTEST = go test -v
GOBUILD = go build -v $(GO_BUILD_FLAGS)

export GO111MODULE=off

all: build

build:
	${GoPath} ${GOBUILD} -o ${BINARY_NAME}

test:
	${GoPath} ${GOTEST} ./...

install:
	install -Dm0755 ll-pica ${DESTDIR}/${PREFIX}/bin/ll-pica

clean:
	rm -rf ${BINARY_NAME}

.PHONY: ${BINARY_NAME}
