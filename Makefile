VET_REPORT = vet.report
TEST_REPORT = tests.xml
GOARCH = amd64

PACKAGE=github.com/argoproj/argo
BUILD_DIR=${GOPATH}/src/${PACKAGE}
DIST_DIR=${GOPATH}/src/${PACKAGE}/dist
CURRENT_DIR=$(shell pwd)

VERSION=$(shell cat ${BUILD_DIR}/VERSION)
COMMIT=$(shell git rev-parse HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

LDFLAGS = -ldflags "-X ${PACKAGE}.Version=${VERSION} -X ${PACKAGE}.Revision=${COMMIT} -X ${PACKAGE}.Branch=${BRANCH}"

BUILDER_IMAGE=argo-builder
BUILDER_CMD=docker run --rm -v ${BUILD_DIR}:/root/go/src/${PACKAGE} -w /root/go/src/${PACKAGE} ${BUILDER_IMAGE}

# Build the project
all: lint cli-linux cli-darwin workflow workflow-image apiserver

builder:
	cd ${BUILD_DIR}; \
	docker build -t ${BUILDER_IMAGE} -f Dockerfile-builder . ; \
	cd - >/dev/null

cli:
	cd ${BUILD_DIR}; \
	go build -v -i ${LDFLAGS} -o ${DIST_DIR}/argo ./cli ; \
	cd - >/dev/null

cli-linux: builder
	rm -f ${DIST_DIR}/argocli/linux-amd64/argo
	${BUILDER_CMD} make cli
	mkdir -p ${DIST_DIR}/argocli/linux-amd64
	mv ${DIST_DIR}/argo ${DIST_DIR}/argocli/linux-amd64/argo

cli-darwin:
	cd ${BUILD_DIR}; \
	GOOS=darwin GOARCH=${GOARCH} go build -v ${LDFLAGS} -o ${DIST_DIR}/argocli/${GOOS}-${GOARCH}/argo ./cli ; \
	cd - >/dev/null

apiserver:
	cd ${BUILD_DIR}; \
	go build -i ${LDFLAGS} -o ${DIST_DIR}/argo-apiserver ./apiserver ; \
	cd - >/dev/null

apiserver-linux: builder
	${BUILDER_CMD} make apiserver

apiserver-image: apiserver-linux
	cd ${BUILD_DIR}; \
	docker build -f Dockerfile-apiserver . ; \
	cd - >/dev/null

workflow:
	cd ${BUILD_DIR}; \
	go build -v -i ${LDFLAGS} -o ${DIST_DIR}/workflow-controller ./workflow ; \
	cd - >/dev/null

workflow-linux: builder
	${BUILDER_CMD} make workflow

workflow-image: workflow-linux
	cd ${BUILD_DIR}; \
	docker build -f Dockerfile-workflow-controller . ; \
	cd - >/dev/null

test:
	if ! hash go2xunit 2>/dev/null; then go install github.com/tebeka/go2xunit; fi
	cd ${BUILD_DIR}; \
	godep go test -v ./... 2>&1 | go2xunit -output ${TEST_REPORT} ; \
	cd - >/dev/null

lint:
	cd ${BUILD_DIR}; \
	gometalinter --config gometalinter.json --deadline 2m --exclude=vendor ./.. ; \
	cd - >/dev/null

fmt:
	cd ${BUILD_DIR}; \
	go fmt $$(go list ./... | grep -v /vendor/) ; \
	cd - >/dev/null

clean:
	-rm -rf ${BUILD_DIR}/dist

.PHONY: builder cli cli-linux cli-darwin workflow workflow-image apiserver lint test fmt clean
