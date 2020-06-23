SHELL=/bin/bash -o pipefail

OUTPUT_IMAGE_OS ?= linux
OUTPUT_IMAGE_ARCH ?= amd64

BUILD_DATE             = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT             = $(shell git rev-parse HEAD)
GIT_REMOTE             = origin
GIT_BRANCH             = $(shell git rev-parse --abbrev-ref=loose HEAD | sed 's/heads\///')
GIT_TAG                = $(shell git describe --exact-match --tags HEAD 2>/dev/null || git rev-parse --short=8 HEAD 2>/dev/null)
GIT_TREE_STATE         = $(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)

export DOCKER_BUILDKIT = 1

# To allow you to build with or without cache for debugging purposes.
DOCKER_BUILD_OPTS     := --no-cache
# Use a different Dockerfile, e.g. for building for Windows or dev images.
DOCKERFILE            := Dockerfile


# docker image publishing options
IMAGE_NAMESPACE       ?= argoproj
# The name of the namespace where Kubernetes resources/RBAC will be installed
KUBE_NAMESPACE        ?= argo

# The rules for what version are, in order of precedence
# 1. If anything passed at the command line (e.g. make release VERSION=...)
# 2. If on master, it must be "latest".
# 3. If on tag, must be tag.
# 4. If on a release branch, the most recent tag that contain the major minor on that branch,
# 5. Otherwise, the branch.
#
VERSION := $(subst /,-,$(GIT_BRANCH))

ifeq ($(GIT_BRANCH),master)
VERSION := latest
endif

ifneq ($(findstring release,$(GIT_BRANCH)),)
VERSION := $(shell git tag --points-at=HEAD|grep ^v|head -n1)
endif

# MANIFESTS_VERSION is the version to be used for files in manifests and should always be latests unles we are releasing
# we assume HEAD means you are on a tag
ifeq ($(GIT_BRANCH),HEAD)
VERSION               := $(GIT_TAG)
MANIFESTS_VERSION     := $(VERSION)
DEV_IMAGE             := false
else
ifeq ($(findstring release,$(GIT_BRANCH)),release)
MANIFESTS_VERSION     := $(VERSION)
DEV_IMAGE             := false
else
MANIFESTS_VERSION     := latest
DEV_IMAGE             := true
endif
endif

# If we are building dev images, then we want to use the Docker cache for speed.
ifeq ($(DEV_IMAGE),true)
DOCKER_BUILD_OPTS     :=
DOCKERFILE            := Dockerfile.dev
endif

# version change, so does the file location
MANIFESTS_VERSION_FILE := dist/$(MANIFESTS_VERSION).manifests-version
VERSION_FILE           := dist/$(VERSION).version
CLI_IMAGE_FILE         := dist/cli-image.$(VERSION)
EXECUTOR_IMAGE_FILE    := dist/executor-image.$(VERSION)
CONTROLLER_IMAGE_FILE  := dist/controller-image.$(VERSION)

# perform static compilation
STATIC_BUILD          ?= true
CI                    ?= false
PROFILE               ?= minimal
AUTH_MODE             := hybrid
ifeq ($(PROFILE),sso)
AUTH_MODE             := sso
endif
ifeq ($(CI),true)
AUTH_MODE             := client
endif
K3D                   := $(shell if [ "`which kubectl`" != '' ] && [ "`kubectl config current-context`" = "k3s-default" ]; then echo true; else echo false; fi)
LOG_LEVEL             := debug

ALWAYS_OFFLOAD_NODE_STATUS := false
ifeq ($(PROFILE),mysql)
ALWAYS_OFFLOAD_NODE_STATUS := true
endif
ifeq ($(PROFILE),postgres)
ALWAYS_OFFLOAD_NODE_STATUS := true
endif

ifeq ($(CI),true)
TEST_OPTS := -coverprofile=coverage.out
else
TEST_OPTS :=
endif

override LDFLAGS += \
  -X github.com/argoproj/argo.version=$(VERSION) \
  -X github.com/argoproj/argo.buildDate=${BUILD_DATE} \
  -X github.com/argoproj/argo.gitCommit=${GIT_COMMIT} \
  -X github.com/argoproj/argo.gitTreeState=${GIT_TREE_STATE}

ifeq ($(STATIC_BUILD), true)
override LDFLAGS += -extldflags "-static"
endif

ifneq ($(GIT_TAG),)
override LDFLAGS += -X github.com/argoproj/argo.gitTag=${GIT_TAG}
endif

ARGOEXEC_PKGS    := $(shell echo cmd/argoexec            && go list -f '{{ join .Deps "\n" }}' ./cmd/argoexec/            | grep 'argoproj/argo' | cut -c 26-)
CLI_PKGS         := $(shell echo cmd/argo                && go list -f '{{ join .Deps "\n" }}' ./cmd/argo/                | grep 'argoproj/argo' | cut -c 26-)
CONTROLLER_PKGS  := $(shell echo cmd/workflow-controller && go list -f '{{ join .Deps "\n" }}' ./cmd/workflow-controller/ | grep 'argoproj/argo' | cut -c 26-)
MANIFESTS        := $(shell find manifests          -mindepth 2 -type f)
E2E_MANIFESTS    := $(shell find test/e2e/manifests -mindepth 2 -type f)
E2E_EXECUTOR     ?= pns
# The sort puts _.primary first in the list. 'env LC_COLLATE=C' makes sure underscore comes first in both Mac and Linux.
SWAGGER_FILES    := $(shell find pkg/apiclient -name '*.swagger.json' | env LC_COLLATE=C sort)
MOCK_FILES       := $(shell find persist server workflow -maxdepth 4 -not -path '/vendor/*' -not -path './ui/*' -path '*/mocks/*' -type f -name '*.go')
UI_FILES         := $(shell find ui/src -type f && find ui -maxdepth 1 -type f)

define backup_go_mod
	# Back-up go.*, but only if we have not already done this (because that would suggest we failed mid-codegen and the currenty go.* files are borked).
	@mkdir -p dist
	[ -e dist/go.mod ] || cp go.mod go.sum dist/
endef
define restore_go_mod
	# Restore the back-ups.
	mv dist/go.mod dist/go.sum .
endef
# docker_build,image_name,binary_name,marker_file_name
define docker_build
	# If we're making a dev build, we build this locally (this will be faster due to existing Go build caches).
	if [ $(DEV_IMAGE) = true ]; then $(MAKE) dist/$(2)-$(OUTPUT_IMAGE_OS)-$(OUTPUT_IMAGE_ARCH) && mv dist/$(2)-$(OUTPUT_IMAGE_OS)-$(OUTPUT_IMAGE_ARCH) $(2); fi
	docker build --progress plain $(DOCKER_BUILD_OPTS) -t $(IMAGE_NAMESPACE)/$(1):$(VERSION) --target $(1) -f $(DOCKERFILE) --build-arg IMAGE_OS=$(OUTPUT_IMAGE_OS) --build-arg IMAGE_ARCH=$(OUTPUT_IMAGE_ARCH) .
	if [ $(DEV_IMAGE) = true ]; then mv $(2) dist/$(2)-$(OUTPUT_IMAGE_OS)-$(OUTPUT_IMAGE_ARCH); fi
	if [ $(K3D) = true ]; then k3d import-images $(IMAGE_NAMESPACE)/$(1):$(VERSION); fi
	touch $(3)
endef
define docker_pull
	docker pull $(1)
	if [ $(K3D) = true ]; then k3d import-images $(1); fi
endef

.PHONY: build
build: status clis executor-image controller-image manifests/install.yaml manifests/namespace-install.yaml manifests/quick-start-postgres.yaml manifests/quick-start-mysql.yaml

# https://stackoverflow.com/questions/4122831/disable-make-builtin-rules-and-variables-from-inside-the-make-file
.SUFFIXES:

.PHONY: status
status:
	# GIT_TAG=$(GIT_TAG), GIT_BRANCH=$(GIT_BRANCH), GIT_TREE_STATE=$(GIT_TREE_STATE), MANIFESTS_VERSION=$(MANIFESTS_VERSION), VERSION=$(VERSION), DEV_IMAGE=$(DEV_IMAGE), K3D=$(K3D)

# cli

.PHONY: cli
cli: dist/argo argo-server.crt argo-server.key

ui/dist/node_modules.marker: ui/package.json ui/yarn.lock
	# Get UI dependencies
	@mkdir -p ui/node_modules
ifeq ($(CI),false)
	yarn --cwd ui install
endif
	@mkdir -p ui/dist
	touch ui/dist/node_modules.marker

ui/dist/app/index.html: ui/dist/node_modules.marker $(UI_FILES)
	# Build UI
	@mkdir -p ui/dist/app
ifeq ($(CI),false)
	yarn --cwd ui build
else
	echo "Built without static files" > ui/dist/app/index.html
endif

$(GOPATH)/bin/staticfiles:
	$(call backup_go_mod)
	go get bou.ke/staticfiles
	$(call restore_go_mod)

server/static/files.go: $(GOPATH)/bin/staticfiles ui/dist/app/index.html
	# Pack UI into a Go file.
	$(GOPATH)/bin/staticfiles -o server/static/files.go ui/dist/app

dist/argo-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/argo-darwin-amd64: GOARGS = GOOS=darwin GOARCH=amd64
dist/argo-windows-amd64: GOARGS = GOOS=windows GOARCH=amd64
dist/argo-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64
dist/argo-linux-ppc64le: GOARGS = GOOS=linux GOARCH=ppc64le
dist/argo-linux-s390x: GOARGS = GOOS=linux GOARCH=s390x

dist/argo: server/static/files.go $(CLI_PKGS)
	go build -v -i -ldflags '${LDFLAGS}' -o dist/argo ./cmd/argo

dist/argo-%: server/static/files.go $(CLI_PKGS)
	CGO_ENABLED=0 $(GOARGS) go build -v -i -ldflags '${LDFLAGS}' -o $@ ./cmd/argo

argo-server.crt: argo-server.key

argo-server.key:
	openssl req -x509 -newkey rsa:4096 -keyout argo-server.key -out argo-server.crt -days 365 -nodes -subj /CN=localhost/O=ArgoProj

.PHONY: cli-image
cli-image: $(CLI_IMAGE_FILE)

$(CLI_IMAGE_FILE): $(CLI_PKGS)
	$(call docker_build,argocli,argo,$(CLI_IMAGE_FILE))

.PHONY: clis
clis: dist/argo-linux-amd64 dist/argo-linux-arm64 dist/argo-linux-ppc64le dist/argo-linux-s390x dist/argo-darwin-amd64 dist/argo-windows-amd64 cli-image

.PHONY: controller
controller: dist/workflow-controller

dist/workflow-controller: GOARGS = GOOS= GOARCH=
dist/workflow-controller-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/workflow-controller-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64

dist/workflow-controller: $(CONTROLLER_PKGS)
	go build -v -i -ldflags '${LDFLAGS}' -o $@ ./cmd/workflow-controller

dist/workflow-controller-%: $(CONTROLLER_PKGS)
	CGO_ENABLED=0 $(GOARGS) go build -v -i -ldflags '${LDFLAGS}' -o $@ ./cmd/workflow-controller

.PHONY: controller-image
controller-image: $(CONTROLLER_IMAGE_FILE)

$(CONTROLLER_IMAGE_FILE): $(CONTROLLER_PKGS)
	$(call docker_build,workflow-controller,workflow-controller,$(CONTROLLER_IMAGE_FILE))

# argoexec

dist/argoexec-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/argoexec-windows-amd64: GOARGS = GOOS=windows GOARCH=amd64
dist/argoexec-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64

dist/argoexec-%: $(ARGOEXEC_PKGS)
	CGO_ENABLED=0 $(GOARGS) go build -v -i -ldflags '${LDFLAGS}' -o $@ ./cmd/argoexec

.PHONY: executor-image
executor-image: $(EXECUTOR_IMAGE_FILE)

	# Create executor image
$(EXECUTOR_IMAGE_FILE): $(ARGOEXEC_PKGS)
	$(call docker_build,argoexec,argoexec,$(EXECUTOR_IMAGE_FILE))

# generation

$(GOPATH)/bin/mockery:
	./hack/recurl.sh dist/mockery.tar.gz https://github.com/vektra/mockery/releases/download/v1.1.1/mockery_1.1.1_$(shell uname -s)_$(shell uname -m).tar.gz
	tar zxvf dist/mockery.tar.gz mockery
	chmod +x mockery
	mkdir -p $(GOPATH)/bin
	mv mockery $(GOPATH)/bin/mockery
	mockery -version

.PHONY: mocks
mocks: $(GOPATH)/bin/mockery
	./hack/update-mocks.sh $(MOCK_FILES)

.PHONY: codegen
codegen: status proto swagger mocks docs

.PHONY: proto
proto:
	$(call backup_go_mod)
	# We need the folder for compatibility
	go mod vendor
	# Generate proto
	./hack/generate-proto.sh
	# Updated codegen
	./hack/update-codegen.sh
	$(call restore_go_mod)

# we use a different file to ./VERSION to force updating manifests after a `make clean`
$(MANIFESTS_VERSION_FILE):
	@mkdir -p dist
	touch $(MANIFESTS_VERSION_FILE)

.PHONY: manifests
manifests:
	./hack/update-image-tags.sh manifests/base $(MANIFESTS_VERSION)
	kustomize build --load_restrictor=none manifests/cluster-install | ./hack/auto-gen-msg.sh > manifests/install.yaml
	kustomize build --load_restrictor=none manifests/namespace-install | ./hack/auto-gen-msg.sh > manifests/namespace-install.yaml
	kustomize build --load_restrictor=none manifests/quick-start/minimal | ./hack/auto-gen-msg.sh > manifests/quick-start-minimal.yaml
	kustomize build --load_restrictor=none manifests/quick-start/mysql | ./hack/auto-gen-msg.sh > manifests/quick-start-mysql.yaml
	kustomize build --load_restrictor=none manifests/quick-start/postgres | ./hack/auto-gen-msg.sh > manifests/quick-start-postgres.yaml

# lint/test/etc

$(GOPATH)/bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin v1.23.8

.PHONY: lint
lint: server/static/files.go $(GOPATH)/bin/golangci-lint
	# Tidy Go modules
	go mod tidy
	# Lint Go files
	golangci-lint run --fix --verbose --concurrency 4 --timeout 5m
	# Lint UI files
ifeq ($(CI),false)
	yarn --cwd ui lint
endif

# for local we have a faster target that prints to stdout, does not use json, and can cache because it has no coverage
.PHONY: test
test: server/static/files.go
	@mkdir -p test-results
	go test -v $(TEST_OPTS) `go list ./... | grep -v 'test/e2e'` 2>&1 | tee test-results/test.out

test-results/test-report.json: test-results/test.out
	cat test-results/test.out | go tool test2json > test-results/test-report.json

$(GOPATH)/bin/go-junit-report:
	$(call backup_go_mod)
	go get github.com/jstemmer/go-junit-report
	$(call restore_go_mod)

# note that we do not have a dependency on test.out, we assume you did correctly create this
test-results/junit.xml: $(GOPATH)/bin/go-junit-report test-results/test.out
	cat test-results/test.out | go-junit-report > test-results/junit.xml

$(VERSION_FILE):
	@mkdir -p dist
	touch $(VERSION_FILE)

dist/$(PROFILE).yaml: $(MANIFESTS) $(E2E_MANIFESTS) $(VERSION_FILE)
	kustomize build --load_restrictor=none test/e2e/manifests/$(PROFILE) | sed 's/:$(MANIFESTS_VERSION)/:$(VERSION)/' | sed 's/pns/$(E2E_EXECUTOR)/'  > dist/$(PROFILE).yaml

.PHONY: install
install: dist/$(PROFILE).yaml
ifeq ($(K3D),true)
	k3d start
endif
	cat test/e2e/manifests/argo-ns.yaml | sed 's/argo/$(KUBE_NAMESPACE)/' > dist/argo-ns.yaml
	kubectl apply -f dist/argo-ns.yaml
	kubectl -n $(KUBE_NAMESPACE) apply -l app.kubernetes.io/part-of=argo --prune --force -f dist/$(PROFILE).yaml

.PHONY: pull-build-images
pull-build-images:
	./hack/pull-build-images.sh

.PHONY: argosay
argosay: test/e2e/images/argosay/v2/argosay
	cd test/e2e/images/argosay/v2 && docker build . -t argoproj/argosay:v2
ifeq ($(K3D),true)
	k3d import-images argoproj/argosay:v2
endif
	docker push argoproj/argosay:v2

test/e2e/images/argosay/v2/argosay: $(shell find test/e2e/images/argosay/v2/main -type f)
	cd test/e2e/images/argosay/v2 && GOOS=linux CGO_ENABLED=0 go build -ldflags '-w -s' -o argosay ./main

.PHONY: test-images
test-images:
	$(call docker_pull,argoproj/argosay:v1)
	$(call docker_pull,argoproj/argosay:v2)
	$(call docker_pull,python:alpine3.6)

.PHONY: stop
stop:
	killall argo workflow-controller pf.sh kubectl || true

$(GOPATH)/bin/goreman:
	go get github.com/mattn/goreman

.PHONY: start
start: status stop install controller cli executor-image $(GOPATH)/bin/goreman
	kubectl config set-context --current --namespace=$(KUBE_NAMESPACE)
	kubectl -n $(KUBE_NAMESPACE) wait --for=condition=Ready pod --all -l app --timeout 2m
	./hack/port-forward.sh
	# Check dex, minio, postgres and mysql are in hosts file
ifeq ($(AUTH_MODE),sso)
	grep '127.0.0.1 *dex' /etc/hosts
endif
	grep '127.0.0.1 *minio' /etc/hosts
	grep '127.0.0.1 *postgres' /etc/hosts
	grep '127.0.0.1 *mysql' /etc/hosts
	env ALWAYS_OFFLOAD_NODE_STATUS=$(ALWAYS_OFFLOAD_NODE_STATUS) LOG_LEVEL=$(LOG_LEVEL) VERSION=$(VERSION) AUTH_MODE=$(AUTH_MODE) $(GOPATH)/bin/goreman -set-ports=false -logtime=false start


.PHONY: wait
wait:
	# Wait for workflow controller
	until lsof -i :9090 > /dev/null ; do sleep 10s ; done
	# Wait for Argo Server
	until lsof -i :2746 > /dev/null ; do sleep 10s ; done

define print_env
	export ARGO_SERVER=localhost:2746
	export ARGO_SECURE=true
	export ARGO_INSECURE_SKIP_VERIFY=true
	export ARGO_TOKEN=$(shell ./dist/argo auth token)
endef

# this is a convenience to get the login token, you can use it as follows
#   eval $(make env)
#   argo token
.PHONY: env
env:
	$(call print_env)

.PHONY: logs
logs:
	# Tail logs
	kubectl -n $(KUBE_NAMESPACE) logs -f -l app --max-log-requests 10 --tail 100

.PHONY: postgres-cli
postgres-cli:
	kubectl exec -ti `kubectl get pod -l app=postgres -o name|cut -c 5-` -- psql -U postgres

.PHONY: mysql-cli
mysql-cli:
	kubectl exec -ti `kubectl get pod -l app=mysql -o name|cut -c 5-` -- mysql -u mysql -ppassword argo

.PHONY: test-e2e
test-e2e: test-images cli
	# Run E2E tests
	@mkdir -p test-results
	go test -timeout 15m -v -count 1 -p 1 --short ./test/e2e/... 2>&1 | tee test-results/test.out

.PHONY: test-e2e-cron
test-e2e-cron: test-images cli
	# Run E2E tests
	@mkdir -p test-results
	go test -timeout 5m -v -count 1 -parallel 10 -run CronSuite ./test/e2e 2>&1 | tee test-results/test.out

.PHONY: smoke
smoke: test-images
	# Run smoke tests
	@mkdir -p test-results
	go test -timeout 1m -v -count 1 -p 1 -run SmokeSuite ./test/e2e 2>&1 | tee test-results/test.out

.PHONY: test-api
test-api:
	# Run API tests
	go test -timeout 1m -v -count 1 -p 1 -run ArgoServerSuite ./test/e2e

.PHONY: test-cli
test-cli: cli
	# Run CLI tests
	go test -timeout 2m -v -count 1 -p 1 -run CLISuite ./test/e2e
	go test -timeout 2m -v -count 1 -p 1 -run CLIWithServerSuite ./test/e2e

# clean

.PHONY: clean
clean:
	# Delete build files
	rm -Rf dist/* ui/dist

# swagger

$(GOPATH)/bin/swagger:
	$(call backup_go_mod)
	go get github.com/go-swagger/go-swagger/cmd/swagger@v0.23.0
	$(call restore_go_mod)

.PHONY: swagger
swagger: api/openapi-spec/swagger.json

pkg/apis/workflow/v1alpha1/openapi_generated.go: $(shell find pkg/apis/workflow/v1alpha1 -type f -not -name openapi_generated.go)
	$(call backup_go_mod)
	go install k8s.io/kube-openapi/cmd/openapi-gen
	openapi-gen \
	  --go-header-file ./hack/custom-boilerplate.go.txt \
	  --input-dirs github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
	  --output-package github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
	  --report-filename pkg/apis/api-rules/violation_exceptions.list
	$(call restore_go_mod)

dist/kubernetes.swagger.json:
	./hack/recurl.sh dist/kubernetes.swagger.json https://raw.githubusercontent.com/kubernetes/kubernetes/release-1.15/api/openapi-spec/swagger.json

pkg/apiclient/_.secondary.swagger.json: hack/secondaryswaggergen.go pkg/apis/workflow/v1alpha1/openapi_generated.go dist/kubernetes.swagger.json
	go run ./hack secondaryswaggergen

# we always ignore the conflicts, so lets automated figuring out how many there will be and just use that
dist/swagger-conflicts: $(GOPATH)/bin/swagger $(SWAGGER_FILES)
	swagger mixin $(SWAGGER_FILES) 2>&1 | grep -c skipping > dist/swagger-conflicts || true

dist/mixed.swagger.json: $(GOPATH)/bin/swagger $(SWAGGER_FILES) dist/swagger-conflicts
	swagger mixin -c $(shell cat dist/swagger-conflicts) $(SWAGGER_FILES) > dist/mixed.swagger.json.tmp
	mv dist/mixed.swagger.json.tmp dist/mixed.swagger.json

dist/swaggifed.swagger.json: dist/mixed.swagger.json $(MANIFESTS_VERSION_FILE) hack/swaggify.sh
	cat dist/mixed.swagger.json | sed 's/VERSION/$(MANIFESTS_VERSION)/' | ./hack/swaggify.sh > dist/swaggifed.swagger.json

dist/kubeified.swagger.json: dist/swaggifed.swagger.json dist/kubernetes.swagger.json hack/kubeifyswagger.go
	go run ./hack kubeifyswagger dist/swaggifed.swagger.json dist/kubeified.swagger.json

api/openapi-spec/swagger.json: dist/kubeified.swagger.json
	swagger flatten --with-flatten minimal --with-flatten remove-unused dist/kubeified.swagger.json > dist/swagger.json
	mv dist/swagger.json api/openapi-spec/swagger.json
	swagger validate api/openapi-spec/swagger.json
	go test ./api/openapi-spec

.PHONY: docs
docs: swagger
	go run ./hack docgen
	go run ./hack readmegen

# pre-push

.PHONY: pre-commit
pre-commit: test lint codegen manifests start smoke test-api test-cli

# release - targets only available on release branch
ifneq ($(findstring release,$(GIT_BRANCH)),)

.PHONY: prepare-release
prepare-release: check-version-warning clean codegen manifests
	# Commit if any changes
	git diff --quiet || git commit -am "Update manifests to $(VERSION)"
	git tag $(VERSION)

.PHONY: publish-release
publish-release: check-version-warning build
	# Push images to Docker Hub
	docker push $(IMAGE_NAMESPACE)/argocli:$(VERSION)
	docker push $(IMAGE_NAMESPACE)/argoexec:$(VERSION)
	docker push $(IMAGE_NAMESPACE)/workflow-controller:$(VERSION)
	git push
	git push $(GIT_REMOTE) $(VERSION)
endif

.PHONY: check-version-warning
check-version-warning:
	@if [[ "$(VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+.*$  ]]; then echo -n "It looks like you're trying to use a SemVer version, but have not prepended it with a "v" (such as "v$(VERSION)"). The "v" is required for our releases. Do you wish to continue anyway? [y/N]" && read ans && [ $${ans:-N} = y ]; fi
