VET_REPORT = vet.report
TEST_REPORT = tests.xml
GOARCH = amd64

PACKAGE=github.com/argoproj/argo
BUILD_DIR=${GOPATH}/src/${PACKAGE}
DIST_DIR=${GOPATH}/src/${PACKAGE}/dist
CURRENT_DIR=$(shell pwd)
BUILD_DIR_LINK=$(shell readlink ${BUILD_DIR})

VERSION=$(shell cat ${BUILD_DIR}/VERSION)
COMMIT=$(shell git rev-parse HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

LDFLAGS = -ldflags "-X ${PACKAGE}.Version=${VERSION} -X ${PACKAGE}.Revision=${COMMIT} -X ${PACKAGE}.Branch=${BRANCH}"

# Build the project
all: lint cli-linux cli-darwin workflow workflow-image

builder:
	cd ${BUILD_DIR}; \
	docker build -t argo-builder -f Dockerfile-builder . ; \
	cd - >/dev/null

cli:
	cd ${BUILD_DIR}; \
	go build -i ${LDFLAGS} -o ${DIST_DIR}/argo ./cli ; \
	cd - >/dev/null

cli-linux: builder
	rm -f ${DIST_DIR}/argocli/linux-amd64/argo
	docker run --rm -v ${BUILD_DIR}:/root/go/src/${PACKAGE} -w /root/go/src/${PACKAGE} argo-builder make cli
	mkdir -p ${DIST_DIR}/argocli/linux-amd64
	mv ${DIST_DIR}/argo ${DIST_DIR}/argocli/linux-amd64/argo

cli-darwin:
	cd ${BUILD_DIR}; \
	GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS} -o ${DIST_DIR}/argocli/darwin-amd64/argo ./cli ; \
	cd - >/dev/null

workflow:
	cd ${BUILD_DIR}; \
	go build -i ${LDFLAGS} -o ${DIST_DIR}/workflow-controller ./workflow ; \
	cd - >/dev/null

workflow-linux: builder
	docker run --rm -v ${BUILD_DIR}:/root/go/src/${PACKAGE} -w /root/go/src/${PACKAGE} argo-builder make workflow

workflow-image: workflow
	cd ${BUILD_DIR}; \
	docker build -f Dockerfile-workflow-controller . ; \
	cd - >/dev/null

link:
	BUILD_DIR=${BUILD_DIR}; \
	BUILD_DIR_LINK=${BUILD_DIR_LINK}; \
	CURRENT_DIR=${CURRENT_DIR}; \
	if [ "$${BUILD_DIR_LINK}" != "$${CURRENT_DIR}" ]; then \
	    echo "Fixing symlinks for build"; \
	    rm -f $${BUILD_DIR}; \
	    ln -s $${CURRENT_DIR} $${BUILD_DIR}; \
	fi

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

.PHONY: builder cli cli-linux cli-darwin workflow workflow-image lint test fmt clean