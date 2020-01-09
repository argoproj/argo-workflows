PACKAGE                := github.com/argoproj/argo

BUILD_DATE             = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT             = $(shell git rev-parse HEAD)
GIT_BRANCH             = $(shell git rev-parse --abbrev-ref HEAD)
GIT_TAG                = $(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
GIT_TREE_STATE         = $(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)

# docker image publishing options
DOCKER_PUSH           ?= false
DOCKER_BUILDKIT       = 1
IMAGE_NAMESPACE       = argoproj

# version must be  branch name or  vX.Y.Z
ifeq ($(GIT_BRANCH), master)
VERSION               ?= latest
else
VERSION               ?= $(GIT_BRANCH)
endif

# perform static compilation
STATIC_BUILD          ?= true
# build development images
DEV_IMAGE             ?= false
CI                    ?= false

override LDFLAGS += \
  -X ${PACKAGE}.version=${VERSION} \
  -X ${PACKAGE}.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}.gitTreeState=${GIT_TREE_STATE}

ifeq (${STATIC_BUILD}, true)
override LDFLAGS += -extldflags "-static"
endif

ifneq (${GIT_TAG},)
VERSION = ${GIT_TAG}
override LDFLAGS += -X ${PACKAGE}.gitTag=${GIT_TAG}
endif

SNAPSHOT=false
ifeq ($(VERSION),latest)
	SNAPSHOT=true
endif
ifeq ($(VERSION),$GIT_BRANCH)
	SNAPSHOT=true
endif


CLI_PKGS := $(shell go list  -f '{{ join .Deps "\n" }}'  ./cmd/argo/|grep 'argoproj/argo'|grep -v vendor|cut -c 26-)
ARGO_SERVER_PKGS := $(shell go list  -f '{{ join .Deps "\n" }}'  ./cmd/server/|grep 'argoproj/argo'|grep -v vendor|cut -c 26-)

# Build the project
.PHONY: all
all: cli controller-image executor-image argo-server

.PHONY:builder-image
builder-image:
	docker build -t $(IMAGE_NAMESPACE)/argo-ci-builder:$(VERSION) --target builder .

ui/node_modules: ui/package.json ui/yarn.lock
ifeq ($(CI),false)
	yarn --cwd ui install --frozen-lockfile --ignore-optional --non-interactive
else
	mkdir -p ui/node_modules
endif
	touch ui/node_modules

ui/dist/app: ui/node_modules ui/src
ifeq ($(CI),false)
	yarn --cwd ui build
else
	mkdir -p ui/dist/app
endif
	touch ui/dist/app

vendor: Gopkg.toml
	dep ensure -v -vendor-only

$(GOPATH)/bin/staticfiles:
	go get bou.ke/staticfiles

cmd/server/static/files.go: ui/dist/app $(GOPATH)/bin/staticfiles
	staticfiles -o cmd/server/static/files.go ui/dist/app

.PHONY: cli
cli: dist/argo

dist/argo: vendor $(CLI_PKGS)
	go build -v -i -ldflags '${LDFLAGS}' -o dist/argo ./cmd/argo

dist/argo-linux-amd64: vendor $(CLI_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-linux-amd64 ./cmd/argo

dist/argo-linux-ppc64le: vendor $(CLI_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-linux-ppc64le ./cmd/argo

dist/argo-linux-s390x: vendor $(CLI_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-linux-s390x ./cmd/argo

dist/argo-darwin-amd64: vendor $(CLI_PKGS)
	CGO_ENABLED=0 GOOS=darwin go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-darwin-amd64 ./cmd/argo

dist/argo-windows-amd64: vendor $(CLI_PKGS)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-windows-amd64 ./cmd/argo

.PHONY: cli-image
cli-image:
	docker build -t $(IMAGE_NAMESPACE)/argocli:$(VERSION) --target argocli .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_NAMESPACE)/argocli:$(VERSION) ; fi

.PHONY: clis
clis: dist/argo-linux-amd64 dist/argo-linux-ppc64le dist/argo-linux-s390x dist/argo-darwin-amd64 dist/argo-windows-amd64 cli-image

.PHONY: controller
controller:
	CGO_ENABLED=0 go build -v -i -ldflags '${LDFLAGS}' -o dist/workflow-controller ./cmd/workflow-controller

.PHONY: controller-image
controller-image:
ifeq ($(DEV_IMAGE), true)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o workflow-controller ./cmd/workflow-controller
	docker build -t $(IMAGE_NAMESPACE)/workflow-controller:$(VERSION) -f Dockerfile.workflow-controller-dev .
	rm -f workflow-controller
else
	docker build -t $(IMAGE_NAMESPACE)/workflow-controller:$(VERSION) --target workflow-controller .
endif
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_NAMESPACE)/workflow-controller:$(VERSION) ; fi

dist/argo-server-linux-amd64: vendor cmd/server/static/files.go $(ARGO_SERVER_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-server-linux-amd64 ./cmd/server

dist/argo-server-linux-ppc64le: vendor cmd/server/static/files.go $(ARGO_SERVER_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-server-linux-ppc64le ./cmd/server

dist/argo-server-linux-s390x: vendor cmd/server/static/files.go $(ARGO_SERVER_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-server-linux-s390x ./cmd/server

dist/argo-server-darwin-amd64: vendor cmd/server/static/files.go $(ARGO_SERVER_PKGS)
	CGO_ENABLED=0 GOOS=darwin go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-server-darwin-amd64 ./cmd/server

dist/argo-server-windows-amd64: vendor cmd/server/static/files.go $(ARGO_SERVER_PKGS)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-server-windows-amd64 ./cmd/server

.PHONY: argo-server-image
argo-server-image: dist/argo-server-linux-amd64
	cp dist/argo-server-linux-amd64 argo-server
	docker build -t $(IMAGE_NAMESPACE)/argo-server:$(VERSION) -f Dockerfile.argo-server .
	rm -f argo-server
ifeq ($(DOCKER_PUSH),true)
	docker push $(IMAGE_NAMESPACE)/argo-server:$(VERSION)
endif

.PHONY: argo-server
argo-server: dist/argo-server-linux-amd64 dist/argo-server-linux-ppc64le dist/argo-server-linux-s390x dist/argo-server-darwin-amd64 dist/argo-server-windows-amd64

.PHONY: executor
executor:
	go build -v -i -ldflags '${LDFLAGS}' -o dist/argoexec ./cmd/argoexec

# To speed up local dev, we only create this when a marker file does not exist.
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
executor-image: dist/executor-base-image
ifeq ($(DEV_IMAGE), true)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o argoexec ./cmd/argoexec
	docker build -t $(IMAGE_NAMESPACE)/argoexec:$(VERSION) -f Dockerfile.argoexec-dev .
	rm -f argoexec
else
	docker build -t $(IMAGE_NAMESPACE)/argoexec:$(VERSION) --target argoexec .
endif
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_NAMESPACE)/argoexec:$(VERSION) ; fi

.PHONY: lint
lint: cmd/server/static/files.go
	golangci-lint run --fix --verbose --config golangci.yml
ifeq ($(CI),false)
	yarn --cwd ui lint
endif

.PHONY: test
test: cmd/server/static/files.go vendor
	go test -covermode=count -coverprofile=coverage.out `go list ./... | grep -v 'test/e2e'`

.PHONY: cover
cover:
	go tool cover -html=coverage.out

.PHONY: codegen
codegen:
	./hack/generate-proto.sh
	./hack/update-codegen.sh
	./hack/update-openapigen.sh
	go run ./hack/gen-openapi-spec/main.go ${VERSION} > ./api/openapi-spec/swagger.json

.PHONY: verify-codegen
verify-codegen:
	./hack/verify-codegen.sh
	./hack/update-openapigen.sh --verify-only
	mkdir -p ./dist
	go run ./hack/gen-openapi-spec/main.go ${VERSION} > ./dist/swagger.json
	diff ./dist/swagger.json ./api/openapi-spec/swagger.json

.PHONY: manifests
manifests:
	./hack/update-manifests.sh

.PHONY: start
start:
	env INSTALL_CLI=0 VERSION=dev ./install.sh
	# Scale down in preparation for re-configuration.
	make down
	# Change to use a "dev" tag and enable debug logging.
	kubectl -n argo patch deployment/workflow-controller --type json --patch '[{"op": "replace", "path": "/spec/template/spec/containers/0/imagePullPolicy", "value": "Never"}, {"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "argoproj/workflow-controller:dev"}, {"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": ["--loglevel", "debug", "--executor-image", "argoproj/argoexec:dev", "--executor-image-pull-policy", "Never"]}]'
	# Turn on the workflow complession feature as much as possible, hopefully to shake out some bugs.
	# kubectl -n argo patch deployment/workflow-controller --type json --patch '[{"op": "add", "path": "/spec/template/spec/containers/0/env", "value": [{"name": "MAX_WORKFLOW_SIZE", "value": "1000"}]}]'
	kubectl -n argo patch deployment/argo-server --type json --patch '[{"op": "replace", "path": "/spec/template/spec/containers/0/imagePullPolicy", "value": "Never"}, {"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "argoproj/argo-server:dev"}, {"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": ["--loglevel", "debug", "--auth-type", "client"]}]'
	# Build controller and executor images.
	make controller-image argo-server-image executor-image DEV_IMAGE=true VERSION=dev
	# Scale up.
	make up
	# Make the CLI
	make cli
	# Wait for apps to be ready.
	kubectl -n argo wait --for=condition=Ready pod --all -l app --timeout 90s
	# Switch to "argo" ns.
	kubectl config set-context --current --namespace=argo

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

.PHONY: postgres-cli
postgres-cli:
	kubectl exec -ti `kubectl get pod -l app=postgres -o name|cut -c 5-` -- psql -U postgres

.PHONY: mysql-cli
mysql-cli:
	kubectl exec -ti `kubectl get pod -l app=mysql -o name|cut -c 5-` -- mysql -u mysql -ppassword argo

.PHONY: test-e2e
test-e2e:
	go test -timeout 20m -v -count 1 -p 1 ./test/e2e/...

.PHONY: clean
clean:
	git clean -fxd -e .idea -e vendor -e ui/node_modules

.PHONY: precheckin
precheckin: test lint verify-codegen

.PHONY: release-prepare
release-prepare:
	@if [ "$(GIT_TREE_STATE)" != "clean" ]; then echo 'git tree state is $(GIT_TREE_STATE)' ; exit 1; fi
ifeq ($(VERSION),)
	echo "unable to prepare release - VERSION undefined"
	exit 1
endif
ifeq ($(GIT_BRANCH),master)
	echo "no release preparation needed for master branch"
else
	echo "preparing release $(VERSION)"
	echo $(VERSION) | cut -c 1- > VERSION
	make manifests VERSION=$(VERSION)
	# only commit if changes
	git diff --quiet || git commit -am "Update manifests to $(VERSION)"
ifneq ($(SNAPSHOT),false)
	git tag $(VERSION)
endif
endif

.PHONY: pre-release-check
pre-release-check: precheckin
	@if [ "$(GIT_TREE_STATE)" != "clean" ]; then echo 'git tree state is $(GIT_TREE_STATE)' ; exit 1; fi
	@if [ -z "$(GIT_TAG)" ]; then echo 'commit must be tagged to perform release' ; exit 1; fi
	@if [ "$(GIT_TAG)" != "v$(VERSION)" ]; then echo 'git tag ($(GIT_TAG)) does not match VERSION (v$(VERSION))'; exit 1; fi

.PHONY: release
release: pre-release-check clis controller-image executor-image argo-server
