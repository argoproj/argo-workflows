PACKAGE                = github.com/argoproj/argo
CURRENT_DIR            = $(shell pwd)
DIST_DIR               = ${CURRENT_DIR}/dist
ARGO_CLI_NAME          = argo

VERSION                = $(shell cat ${CURRENT_DIR}/VERSION)
BUILD_DATE             = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT             = $(shell git rev-parse HEAD)
GIT_TAG                = $(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
GIT_TREE_STATE         = $(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)

# docker image publishing options
DOCKER_PUSH           ?= false
IMAGE_TAG             ?= latest
# perform static compilation
STATIC_BUILD          ?= true
# build development images
DEV_IMAGE             ?= false

GOLANGCI_EXISTS := $(shell command -v golangci-lint 2> /dev/null)

override LDFLAGS += \
  -X ${PACKAGE}.version=${VERSION} \
  -X ${PACKAGE}.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}.gitTreeState=${GIT_TREE_STATE}

ifeq (${STATIC_BUILD}, true)
override LDFLAGS += -extldflags "-static"
endif

ifneq (${GIT_TAG},)
IMAGE_TAG = ${GIT_TAG}
override LDFLAGS += -X ${PACKAGE}.gitTag=${GIT_TAG}
endif

ifeq (${DOCKER_PUSH}, true)
ifndef IMAGE_NAMESPACE
$(error IMAGE_NAMESPACE must be set to push images (e.g. IMAGE_NAMESPACE=argoproj))
endif
endif

ifdef IMAGE_NAMESPACE
IMAGE_PREFIX = ${IMAGE_NAMESPACE}/
endif

# Build the project
.PHONY: all
all: cli controller-image executor-image

.PHONY: builder-image
builder-image:
	docker build -t $(IMAGE_PREFIX)argo-ci-builder:$(IMAGE_TAG) --target builder .

.PHONY: cli
cli:
	go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/${ARGO_CLI_NAME} ./cmd/argo

.PHONY: cli-linux-amd64
cli-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argo-linux-amd64 ./cmd/argo

.PHONY: cli-linux-ppc64le
cli-linux-ppc64le:
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argo-linux-ppc64le ./cmd/argo

.PHONY: cli-linux-s390x
cli-linux-s390x:
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argo-linux-s390x ./cmd/argo

.PHONY: cli-linux
cli-linux: cli-linux-amd64 cli-linux-ppc64le cli-linux-s390x

.PHONY: cli-darwin
cli-darwin:
	CGO_ENABLED=0 GOOS=darwin go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argo-darwin-amd64 ./cmd/argo

.PHONY: cli-windows
cli-windows:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argo-windows-amd64 ./cmd/argo

.PHONY: cli-image
cli-image:
	docker build -t $(IMAGE_PREFIX)argocli:$(IMAGE_TAG) --target argocli .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)argocli:$(IMAGE_TAG) ; fi

.PHONY: controller
controller:
	CGO_ENABLED=0 go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/workflow-controller ./cmd/workflow-controller

.PHONY: controller-image
controller-image:
ifeq ($(DEV_IMAGE), true)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o workflow-controller ./cmd/workflow-controller
	docker build -t $(IMAGE_PREFIX)workflow-controller:$(IMAGE_TAG) -f Dockerfile.workflow-controller-dev .
	rm -f workflow-controller
else
	docker build -t $(IMAGE_PREFIX)workflow-controller:$(IMAGE_TAG) --target workflow-controller .
endif
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)workflow-controller:$(IMAGE_TAG) ; fi

.PHONY: executor
executor:
	go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argoexec ./cmd/argoexec

.PHONY: executor-base-image
executor-base-image:
	docker build -t argoexec-base --target argoexec-base .

# The DEV_IMAGE versions of controller-image and executor-image are speed optimized development
# builds of workflow-controller and argoexec images respectively. It allows for faster image builds
# by re-using the golang build cache of the desktop environment. Ideally, we would not need extra
# Dockerfiles for these, and the targets would be defined as new targets in the main Dockerfile, but
# intelligent skipping of docker build stages requires DOCKER_BUILDKIT=1 enabled, which not all
# docker daemons support (including the daemon currently used by minikube).
# TODO: move these targets to the main Dockerfile once DOCKER_BUILDKIT=1 is more pervasive.
# NOTE: have to output ouside of dist directory since dist is under .dockerignore
.PHONY: executor-image
ifeq ($(DEV_IMAGE), true)
executor-image: executor-base-image
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o argoexec ./cmd/argoexec
	docker build -t $(IMAGE_PREFIX)argoexec:$(IMAGE_TAG) -f Dockerfile.argoexec-dev .
	rm -f argoexec
else
executor-image:
	docker build -t $(IMAGE_PREFIX)argoexec:$(IMAGE_TAG) --target argoexec .
endif
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)argoexec:$(IMAGE_TAG) ; fi

.PHONY: lint
lint:
ifdef GOLANGCI_EXISTS
	golangci-lint run --config golangci.yml
else
	# Remove gometalinter after a migration time.
	gometalinter --config gometalinter.json ./...
endif

.PHONY: test
test:
	go test -covermode=count -coverprofile=coverage.out ./...

.PHONY: cover
cover:
	go tool cover -html=coverage.out

.PHONY: codegen
codegen:
	./hack/update-codegen.sh
	./hack/update-openapigen.sh
	go run ./hack/gen-openapi-spec/main.go ${VERSION} > ${CURRENT_DIR}/api/openapi-spec/swagger.json

.PHONY: verify-codegen
verify-codegen:
	./hack/verify-codegen.sh
	./hack/update-openapigen.sh --verify-only
	mkdir -p ${CURRENT_DIR}/dist
	go run ./hack/gen-openapi-spec/main.go ${VERSION} > ${CURRENT_DIR}/dist/swagger.json
	diff ${CURRENT_DIR}/dist/swagger.json ${CURRENT_DIR}/api/openapi-spec/swagger.json

.PHONY: manifests
manifests:
	./hack/update-manifests.sh

.PHONY: clean
clean:
	-rm -rf ${CURRENT_DIR}/dist

.PHONY: precheckin
precheckin: test lint verify-codegen

.PHONY: release-precheck
release-precheck: manifests codegen precheckin
	@if [ "$(GIT_TREE_STATE)" != "clean" ]; then echo 'git tree state is $(GIT_TREE_STATE)' ; exit 1; fi
	@if [ -z "$(GIT_TAG)" ]; then echo 'commit must be tagged to perform release' ; exit 1; fi
	@if [ "$(GIT_TAG)" != "v$(VERSION)" ]; then echo 'git tag ($(GIT_TAG)) does not match VERSION (v$(VERSION))'; exit 1; fi

.PHONY: release-clis
release-clis: cli-image
	docker build --iidfile /tmp/argo-cli-build --target argo-build --build-arg MAKE_TARGET="cli-darwin cli-windows" .
	docker create --name tmp-cli `cat /tmp/argo-cli-build`
	mkdir -p ${DIST_DIR}
	docker cp tmp-cli:/go/src/github.com/argoproj/argo/dist/argo-darwin-amd64 ${DIST_DIR}/argo-darwin-amd64
	docker cp tmp-cli:/go/src/github.com/argoproj/argo/dist/argo-windows-amd64 ${DIST_DIR}/argo-windows-amd64
	docker rm tmp-cli
	docker create --name tmp-cli $(IMAGE_PREFIX)argocli:$(IMAGE_TAG)
	docker cp tmp-cli:/bin/argo ${DIST_DIR}/argo-linux-amd64
	docker rm tmp-cli

.PHONY: release
release: release-precheck controller-image executor-image cli-image release-clis
