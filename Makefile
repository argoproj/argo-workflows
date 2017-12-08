GOARCH=amd64
GOPATH=$(shell go env GOPATH)

PACKAGE=github.com/argoproj/argo
BUILD_DIR=${GOPATH}/src/${PACKAGE}
DIST_DIR=${GOPATH}/src/${PACKAGE}/dist
CURRENT_DIR=$(shell pwd)

VERSION=$(shell cat ${BUILD_DIR}/VERSION)
REVISION=$(shell git rev-parse HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
TAG=$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
TREE_STATE=$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)

BUILDER_IMAGE=argo-builder
BUILDER_CMD=docker run --rm \
  -v ${BUILD_DIR}:/root/go/src/${PACKAGE} \
  -v ${GOPATH}/pkg:/root/go/pkg \
  -w /root/go/src/${PACKAGE} ${BUILDER_IMAGE}

# docker image publishing options
DOCKER_PUSH=false
ifeq (${IMAGE_TAG},)
ifneq (${TAG},)
IMAGE_TAG=${TAG}
else
IMAGE_TAG=${VERSION}
endif
endif

LDFLAGS = -ldflags "-X ${PACKAGE}.Version=${VERSION} \
  -X ${PACKAGE}.Revision=${REVISION} \
  -X ${PACKAGE}.Branch=${BRANCH} \
  -X ${PACKAGE}.Tag=${TAG} \
  -X ${PACKAGE}.ImageNamespace=${IMAGE_NAMESPACE} \
  -X ${PACKAGE}.ImageTag=${IMAGE_TAG}"

ifeq (${DOCKER_PUSH},true)
ifndef IMAGE_NAMESPACE
$(error IMAGE_NAMESPACE must be set to push images (e.g. IMAGE_NAMESPACE=argoproj))
endif
endif

ifdef IMAGE_NAMESPACE
IMAGE_PREFIX=${IMAGE_NAMESPACE}/
endif

# Build the project
all: no_ui ui-image

no_ui: cli controller-image executor-image

builder:
	docker build -t ${BUILDER_IMAGE} -f Dockerfile-builder .

cli:
	go build -v -i ${LDFLAGS} -o ${DIST_DIR}/argo ./cmd/argo

cli-linux: builder
	${BUILDER_CMD} make cli IMAGE_TAG=$(IMAGE_TAG)
	mv ${DIST_DIR}/argo ${DIST_DIR}/argo-linux-amd64

cli-darwin: builder
	${BUILDER_CMD} make cli GOOS=darwin IMAGE_TAG=$(IMAGE_TAG)
	mv ${DIST_DIR}/argo ${DIST_DIR}/argo-darwin-amd64

controller:
	go build -v -i ${LDFLAGS} -o ${DIST_DIR}/workflow-controller ./cmd/workflow-controller

controller-linux: builder
	${BUILDER_CMD} make controller

controller-image: controller-linux
	docker build -t $(IMAGE_PREFIX)workflow-controller:$(IMAGE_TAG) -f Dockerfile-workflow-controller .
	if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)workflow-controller:$(IMAGE_TAG) ; fi

executor:
	go build -i ${LDFLAGS} -o ${DIST_DIR}/argoexec ./cmd/argoexec

executor-linux: builder
	${BUILDER_CMD} make executor

executor-image: executor-linux
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

ui-image:
	docker build -t $(IMAGE_PREFIX)argoui:$(IMAGE_TAG) -f ui/Dockerfile ui
	if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)argoui:$(IMAGE_TAG) ; fi

release-precheck:
	@if [ "$(TREE_STATE)" != "clean" ]; then echo 'git tree state is $(TREE_STATE)' ; exit 1; fi
	@if [ -z "$(TAG)" ]; then echo 'commit must be tagged to perform release' ; exit 1; fi

release: release-precheck controller-image cli-darwin cli-linux executor-image ui-image

.PHONY: builder \
	cli cli-linux cli-darwin \
	controller controller-linux controller-image \
	executor executor-linux executor-image \
	ui-image \
	release-precheck release \
	lint
	# test fmt clean
