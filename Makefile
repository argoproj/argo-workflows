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
TAG=$(shell git describe --exact-match --tags HEAD 2>/dev/null)

BUILDER_IMAGE=argo-builder
BUILDER_CMD=docker run --rm \
  -v ${BUILD_DIR}:/root/go/src/${PACKAGE} \
  -v ${GOPATH}/pkg:/root/go/pkg \
  -w /root/go/src/${PACKAGE} ${BUILDER_IMAGE}

# docker image publishing options
DOCKER_PUSH=false
IMAGE_TAG=${VERSION}

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
all: cli controller-image executor-image ui-image

builder:
	docker build -t ${BUILDER_IMAGE} -f Dockerfile-builder .

cli:
	go build -v -i ${LDFLAGS} -o ${DIST_DIR}/argo ./cmd/argo

cli-linux: builder
	rm -f ${DIST_DIR}/argocli/linux-amd64/argo
	${BUILDER_CMD} make cli
	mkdir -p ${DIST_DIR}/argocli/linux-amd64
	mv ${DIST_DIR}/argo ${DIST_DIR}/argocli/linux-amd64/argo

cli-darwin: builder
	rm -f ${DIST_DIR}/argocli/darwin-amd64/argo
	${BUILDER_CMD} make cli GOOS=darwin
	mkdir -p ${DIST_DIR}/argocli/darwin-amd64
	mv ${DIST_DIR}/argo ${DIST_DIR}/argocli/darwin-amd64/argo

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
	docker run --rm -v `pwd`/ui:/src -w /src -it node:6.9.5 bash -c "npm install -g yarn && rm -rf node_modules && yarn install && yarn run build" && \
	docker build -t $(IMAGE_PREFIX)argoui:$(IMAGE_TAG) -f ui/Dockerfile ui
	if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)argoui:$(IMAGE_TAG) ; fi

.PHONY: builder \
	cli cli-linux cli-darwin \
	controller controller-linux controller-image \
	executor executor-linux executor-image \
	lint
	# test fmt clean
