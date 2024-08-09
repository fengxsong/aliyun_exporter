SHORT_NAME ?= aliyun_exporter

BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
HASH = $(shell git describe --dirty --tags --always)
REF_NAME = $(shell git rev-parse --abbrev-ref HEAD)
BUILDUSER = $(shell whoami)
VERSION ?= unknown
REPO = github.com/fengxsong/aliyun_exporter

BUILD_PATH = main.go
OUTPUT_PATH = build/_output/bin/$(SHORT_NAME)

LDFLAGS := -s -X github.com/prometheus/common/version.Version=${VERSION} \
	-X github.com/prometheus/common/version.Revision=${HASH} \
	-X github.com/prometheus/common/version.Branch=${REF_NAME} \
	-X github.com/prometheus/common/version.BuildUser=${BUILDUSER} \
	-X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}

deps:
	go mod tidy

bin:
	CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o ${OUTPUT_PATH} ${BUILD_PATH} || exit 1

linux-bin:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o ${OUTPUT_PATH} ${BUILD_PATH} || exit 1

upx:
	upx ${OUTPUT_PATH}
