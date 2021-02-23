SHORT_NAME ?= aliyun-exporter

BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
HASH = $(shell git describe --dirty --tags --always)
VERSION ?= unknown
REPO = github.com/fengxsong/aliyun-exporter

BUILD_PATH = main.go
OUTPUT_PATH = build/_output/bin/$(SHORT_NAME)

LDFLAGS := -s -X ${REPO}/version.buildDate=${BUILD_DATE} \
	-X ${REPO}/version.gitCommit=${HASH} \
	-X ${REPO}/version.version=${VERSION}

IMAGE_REPO ?= fengxsong/${SHORT_NAME}
IMAGE_TAG ?= ${HASH}
IMAGE := ${IMAGE_REPO}:${IMAGE_TAG}

tidy:
	go mod tidy

vendor: tidy
	go mod vendor

bin:
	CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o ${OUTPUT_PATH} ${BUILD_PATH} || exit 1

linux-bin:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o ${OUTPUT_PATH} ${BUILD_PATH} || exit 1

upx:
	upx ${OUTPUT_PATH}

# Build the docker image
docker-build:
	docker build --rm --build-arg=LDFLAGS="${LDFLAGS}" -t ${IMAGE} -t ${IMAGE_REPO}:latest -f Dockerfile .

# Push the docker image
docker-push:
	docker push ${IMAGE}
