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
# build static files, disable if you don't need HTML files, e.g. when on CI
STATIC                ?= true

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
all: cli controller-image executor-image argo-server

.PHONY:builder-image
builder-image:
	docker build -t $(IMAGE_PREFIX)argo-ci-builder:$(IMAGE_TAG) --target builder .

ui/dist/app:
ifeq ($(STATIC), true)
	sh -c 'cd ui && make'
else
	mkdir -p ui/dist/app
	echo "UI was disabled in the build" > ui/dist/app/index.html
endif

cmd/server/static/files.go: ui/dist/app
	go get bou.ke/staticfiles
	staticfiles -o cmd/server/static/files.go ui/dist/app

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

.PHONY: argo-server
argo-server: cmd/server/static/files.go
	CGO_ENABLED=0 go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argo-server ./cmd/server

.PHONY: argo-server-image
argo-server-image: cmd/server/static/files.go
ifeq ($(DEV_IMAGE), true)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o argo-server ./cmd/server
	docker build -t $(IMAGE_PREFIX)argo-server:$(IMAGE_TAG) -f Dockerfile.argo-server-dev .
	rm -f argo-server
else
	docker build -t $(IMAGE_PREFIX)argo-server:$(IMAGE_TAG) --target argo-server .
endif
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)argo-server:$(IMAGE_TAG) ; fi

.PHONY: argo-server-linux-amd64
argo-server-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argo-server-linux-amd64 ./cmd/server

.PHONY: argo-server-linux-ppc64le
argo-server-linux-ppc64le:
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argo-server-linux-ppc64le ./cmd/server

.PHONY: argo-server-linux-s390x
argo-server-linux-s390x:
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argo-server-linux-s390x ./cmd/server

.PHONY: argo-server-linux
argo-server-linux: argo-server-linux-amd64 argo-server-linux-ppc64le argo-server-linux-s390x

.PHONY: argo-server-darwin
argo-server-darwin:
	CGO_ENABLED=0 GOOS=darwin go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argo-server-darwin-amd64 ./cmd/server

.PHONY: argo-server-windows
argo-server-windows:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argo-server-windows-amd64 ./cmd/server

.PHONY: executor
executor:
	go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/argoexec ./cmd/argoexec

# To speed up local dev, we only create this when a marker file does not exist.
executor-base-image: dist/executor-base-image
dist/executor-base-image:
	docker build -t argoexec-base --target argoexec-base .
	mkdir -p dist
	touch dist/executor-base-image

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
executor-image: executor-base-image
	docker build -t $(IMAGE_PREFIX)argoexec:$(IMAGE_TAG) --target argoexec .
endif
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)argoexec:$(IMAGE_TAG) ; fi

.PHONY: lint
lint:
	golangci-lint run --fix --verbose --config golangci.yml

.PHONY: test
test: cmd/server/static/files.go
	go test -covermode=count -coverprofile=coverage.out `go list ./... | grep -v e2e`

.PHONY: cover
cover:
	go tool cover -html=coverage.out

.PHONY: codegen
codegen:
	./hack/generate-proto.sh
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

.PHONY: start
start:
	kubectl create ns argo || true
	# Install the standard Argo.
	kubectl -n argo apply --wait --force -f manifests/install.yaml
	# Scale down in preparation for re-configuration.
	make down
	# Change to use a "dev" tag and enable debug logging.
	kubectl -n argo patch deployment/workflow-controller --type json --patch '[{"op": "replace", "path": "/spec/template/spec/containers/0/imagePullPolicy", "value": "Never"}, {"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "argoproj/workflow-controller:dev"}, {"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": ["--loglevel", "debug", "--executor-image", "argoproj/argoexec:dev", "--executor-image-pull-policy", "Never"]}]'
	kubectl -n argo patch deployment/argo-server --type json --patch '[{"op": "replace", "path": "/spec/template/spec/containers/0/imagePullPolicy", "value": "Never"}, {"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "argoproj/argo-server:dev"}, {"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": ["--loglevel", "debug", "--enable-client-auth"]}]'
	# Install MinIO and set-up config-map.
	kubectl -n argo apply --wait --force -f test/e2e/manifests
	# Build controller and executor images.
	make controller-image argo-server-image executor-image DEV_IMAGE=true IMAGE_PREFIX=argoproj/ IMAGE_TAG=dev
	# Scale up.
	make up
	# Wait for apps to be ready.
	kubectl -n argo wait --for=condition=Ready pod --all -l app --timeout 90s
	# Switch to "argo" ns.
	kubectl config set-context --current --namespace=argo
	# Update the config.
	./hack/update-in-cluster-config.sh

.PHONY: down
down:
	kubectl -n argo scale deployment/argo-server --replicas 0
	kubectl -n argo scale deployment/workflow-controller --replicas 0

.PHONY: up
up:
	kubectl -n argo scale deployment/workflow-controller --replicas 1
	kubectl -n argo scale deployment/argo-server --replicas 1

.PHONY: pf
pf:
	./hack/port-forward.sh

.PHONY: logs
logs:
	kubectl -n argo logs -f -l app --max-log-requests 10

.PHONY: test-e2e
test-e2e:
	go test -v -count 1 -p 1 ./test/e2e

.PHONY: clean
clean:
	git clean -fxd

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
