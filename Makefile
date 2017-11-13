GOARCH=amd64
GOPATH=$(shell go env GOPATH)

PACKAGE=github.com/argoproj/argo
BUILD_DIR=${GOPATH}/src/${PACKAGE}
DIST_DIR=${GOPATH}/src/${PACKAGE}/dist
CURRENT_DIR=$(shell pwd)

VERSION=$(shell cat ${BUILD_DIR}/VERSION)
REVISION=$(shell git rev-parse HEAD)
REVISION_SHORT=$(shell git rev-parse --short=7 HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

LDFLAGS = -ldflags "-X ${PACKAGE}.Version=${VERSION} -X ${PACKAGE}.Revision=${REVISION} -X ${PACKAGE}.Branch=${BRANCH}"

BUILDER_IMAGE=argo-builder
BUILDER_CMD=docker run --rm \
  -v ${BUILD_DIR}:/root/go/src/${PACKAGE} \
  -v ${GOPATH}/pkg:/root/go/pkg \
  -w /root/go/src/${PACKAGE} ${BUILDER_IMAGE}

# docker image publishing options
DOCKER_PUSH=false
IMAGE_TAG=${VERSION}-${REVISION_SHORT}

ifeq (${DOCKER_PUSH},true)
ifndef IMAGE_NAMESPACE
$(error IMAGE_NAMESPACE must be set to push images (e.g. IMAGE_NAMESPACE=argoproj))
endif
endif

ifdef IMAGE_NAMESPACE
IMAGE_PREFIX=${IMAGE_NAMESPACE}/
endif

# Build the project
all: cli-linux cli-darwin workflow-image argoexec-image

builder:
	docker build -t ${BUILDER_IMAGE} -f Dockerfile-builder .

cli:
	go build -v -i ${LDFLAGS} -o ${DIST_DIR}/argo ./cmd/argo

cli-linux: builder
	rm -f ${DIST_DIR}/argocli/linux-amd64/argo
	${BUILDER_CMD} make cli
	mkdir -p ${DIST_DIR}/argocli/linux-amd64
	mv ${DIST_DIR}/argo ${DIST_DIR}/argocli/linux-amd64/argo

cli-darwin:
	GOOS=darwin GOARCH=${GOARCH} go build -v ${LDFLAGS} -o ${DIST_DIR}/argocli/${GOOS}-${GOARCH}/argo ./cmd/argo

apiserver:
	go build -v -i ${LDFLAGS} -o ${DIST_DIR}/argo-apiserver ./cmd/argo-apiserver

apiserver-linux: builder
	${BUILDER_CMD} make apiserver

apiserver-image: apiserver-linux
	docker build -t $(IMAGE_PREFIX)workflow-controller:$(IMAGE_TAG) -f Dockerfile-workflow-controller .
	if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)workflow-controller:$(IMAGE_TAG) ; fi

workflow:
	go build -v -i ${LDFLAGS} -o ${DIST_DIR}/workflow-controller ./cmd/workflow-controller

workflow-linux: builder
	${BUILDER_CMD} make workflow

workflow-image: workflow-linux
	docker build -t $(IMAGE_PREFIX)workflow-controller:$(IMAGE_TAG) -f Dockerfile-workflow-controller .
	if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)workflow-controller:$(IMAGE_TAG) ; fi

argoexec:
	go build -i ${LDFLAGS} -o ${DIST_DIR}/argoexec ./cmd/argoexec

argoexec-linux: builder
	${BUILDER_CMD} make argoexec

argoexec-image: argoexec-linux
	docker build -t $(IMAGE_PREFIX)argoexec:$(IMAGE_TAG) -f Dockerfile-argoexec .
	if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)argoexec:$(IMAGE_TAG) ; fi

lint:
	gometalinter --config gometalinter.json --vendor ./...

fmt:
	cd ${BUILD_DIR}; \
	go fmt $$(go list ./... | grep -v /vendor/) ; \
	cd - >/dev/null

clean:
	-rm -rf ${BUILD_DIR}/dist

.PHONY: builder \
	cli cli-linux cli-darwin \
	workflow workflow-linux workflow-image \
	apiserver apiserver-linux apiserver-image \
	argoexec argoexec-linux argoexec-image \
	lint test fmt clean
