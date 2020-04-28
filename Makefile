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

# To allow you to build with cache for debugging purposes.
DOCKER_BUILD_OPTS     := --no-cache

# docker image publishing options
IMAGE_NAMESPACE       ?= argoproj

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
# this will be something like "v2.5" or "v3.7"
MAJOR_MINOR := v$(word 2,$(subst -, ,$(GIT_BRANCH)))
# if GIT_TAG is on HEAD, then this will be the same
GIT_LATEST_TAG := $(shell git tag --merged | tail -n1)
# only use the latest tag if it matches the correct major/minor version
ifneq ($(findstring $(MAJOR_MINOR),$(GIT_LATEST_TAG)),)
VERSION := $(GIT_LATEST_TAG)
endif
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

# version change, so does the file location
MANIFESTS_VERSION_FILE := dist/$(MANIFESTS_VERSION).manifests-version
VERSION_FILE           := dist/$(VERSION).version
CLI_IMAGE_FILE         := dist/cli-image.$(VERSION)
EXECUTOR_IMAGE_FILE    := dist/executor-image.$(VERSION)
CONTROLLER_IMAGE_FILE  := dist/controller-image.$(VERSION)

# perform static compilation
STATIC_BUILD          ?= true
CI                    ?= false
DB                    ?= postgres
K3D                   := $(shell if [ "`kubectl config current-context`" = "k3s-default" ]; then echo true; else echo false; fi)
# which components to start, useful if you want to disable them to debug
COMPONENTS            := controller,argo-server

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
MOCK_FILES       := $(shell find persist workflow -maxdepth 4 -not -path '/vendor/*' -not -path './ui/*' -path '*/mocks/*' -type f -name '*.go')
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

.PHONY: build
build: status clis executor-image controller-image manifests/install.yaml manifests/namespace-install.yaml manifests/quick-start-postgres.yaml manifests/quick-start-mysql.yaml

# https://stackoverflow.com/questions/4122831/disable-make-builtin-rules-and-variables-from-inside-the-make-file
.SUFFIXES:

.PHONY: status
status:
	# GIT_TAG=$(GIT_TAG), GIT_BRANCH=$(GIT_BRANCH), GIT_TREE_STATE=$(GIT_TREE_STATE), MANIFESTS_VERSION=$(MANIFESTS_VERSION), VERSION=$(VERSION), DEV_IMAGE=$(DEV_IMAGE)

# cli

.PHONY: cli
cli: dist/argo argo-server.crt argo-server.key

ui/dist/node_modules.marker: ui/package.json ui/yarn.lock
	# Get UI dependencies
	@mkdir -p ui/node_modules
ifeq ($(CI),false)
	yarn --cwd ui install --frozen-lockfile --ignore-optional --non-interactive
endif
	@mkdir -p ui/dist
	touch ui/dist/node_modules.marker

ui/dist/app/index.html: ui/dist/node_modules.marker ui/src
	# Build UI
	@mkdir -p ui/dist/app
ifeq ($(CI),false)
	yarn --cwd ui build
else
	echo "Built without static files" > ui/dist/app/index.html
endif

$(HOME)/go/bin/staticfiles:
	# Install the "staticfiles" tool
	go get bou.ke/staticfiles

server/static/files.go: $(HOME)/go/bin/staticfiles ui/dist/app/index.html
	# Pack UI into a Go file.
	staticfiles -o server/static/files.go ui/dist/app

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

$(CLI_IMAGE_FILE):
	docker build $(DOCKER_BUILD_OPTS) -t $(IMAGE_NAMESPACE)/argocli:$(VERSION) --target argocli --build-arg IMAGE_OS=$(OUTPUT_IMAGE_OS) --build-arg IMAGE_ARCH=$(OUTPUT_IMAGE_ARCH) .
	touch $(CLI_IMAGE_FILE)

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

$(CONTROLLER_IMAGE_FILE):
	docker build $(DOCKER_BUILD_OPTS) -t $(IMAGE_NAMESPACE)/workflow-controller:$(VERSION) --target workflow-controller --build-arg IMAGE_OS=$(OUTPUT_IMAGE_OS) --build-arg IMAGE_ARCH=$(OUTPUT_IMAGE_ARCH) .
	touch $(CONTROLLER_IMAGE_FILE)

# argoexec

dist/argoexec-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/argoexec-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64

dist/argoexec-%: $(ARGOEXEC_PKGS)
	CGO_ENABLED=0 $(GOARGS) go build -v -i -ldflags '${LDFLAGS}' -o $@ ./cmd/argoexec

.PHONY: executor-image
executor-image: $(EXECUTOR_IMAGE_FILE)

$(EXECUTOR_IMAGE_FILE): dist/argoexec-$(OUTPUT_IMAGE_OS)-$(OUTPUT_IMAGE_ARCH)
	# Create executor image
ifeq ($(DEV_IMAGE),true)
	mv dist/argoexec-$(OUTPUT_IMAGE_OS)-$(OUTPUT_IMAGE_ARCH) argoexec
	docker build -t $(IMAGE_NAMESPACE)/argoexec:$(VERSION) --target argoexec -f Dockerfile.dev --build-arg IMAGE_OS=$(OUTPUT_IMAGE_OS) --build-arg IMAGE_ARCH=$(OUTPUT_IMAGE_ARCH) .
	mv argoexec dist/argoexec-$(OUTPUT_IMAGE_OS)-$(OUTPUT_IMAGE_ARCH)
else
	docker build $(DOCKER_BUILD_OPTS) -t $(IMAGE_NAMESPACE)/argoexec:$(VERSION) --target argoexec --build-arg IMAGE_OS=$(OUTPUT_IMAGE_OS) --build-arg IMAGE_ARCH=$(OUTPUT_IMAGE_ARCH) .
endif
ifeq ($(K3D),true)
	k3d import-images $(IMAGE_NAMESPACE)/argoexec:$(VERSION)
endif
	touch $(EXECUTOR_IMAGE_FILE)

# generation

$(HOME)/go/bin/mockery:
	$(call backup_go_mod)
	go get github.com/vektra/mockery/.../
	$(call restore_go_mod)

.PHONY: mocks
mocks: $(HOME)/go/bin/mockery
	./hack/update-mocks.sh $(MOCK_FILES)

.PHONY: codegen
codegen: status codegen-core swagger mocks docs

.PHONY: codegen-core
codegen-core:
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
	kustomize build --load_restrictor=none manifests/quick-start/no-db | ./hack/auto-gen-msg.sh > manifests/quick-start-no-db.yaml
	kustomize build --load_restrictor=none manifests/quick-start/mysql | ./hack/auto-gen-msg.sh > manifests/quick-start-mysql.yaml
	kustomize build --load_restrictor=none manifests/quick-start/postgres | ./hack/auto-gen-msg.sh > manifests/quick-start-postgres.yaml

# lint/test/etc

$(HOME)/go/bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin v1.23.8

.PHONY: lint
lint: server/static/files.go $(HOME)/go/bin/golangci-lint
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

$(HOME)/go/bin/go-junit-report:
	$(call backup_go_mod)
	go get github.com/jstemmer/go-junit-report
	$(call restore_go_mod)

# note that we do not have a dependency on test.out, we assume you did correctly create this
test-results/junit.xml: $(HOME)/go/bin/go-junit-report test-results/test.out
	cat test-results/test.out | go-junit-report > test-results/junit.xml

$(VERSION_FILE):
	@mkdir -p dist
	touch $(VERSION_FILE)

dist/postgres.yaml: $(MANIFESTS) $(E2E_MANIFESTS) $(VERSION_FILE)
	kustomize build --load_restrictor=none test/e2e/manifests/postgres | sed 's/:$(MANIFESTS_VERSION)/:$(VERSION)/' | sed 's/pns/$(E2E_EXECUTOR)/' > dist/postgres.yaml

dist/no-db.yaml: $(MANIFESTS) $(E2E_MANIFESTS) $(VERSION_FILE)
	# We additionally disable ALWAYS_OFFLOAD_NODE_STATUS
	kustomize build --load_restrictor=none test/e2e/manifests/no-db | sed 's/:$(MANIFESTS_VERSION)/:$(VERSION)/' | sed 's/pns/$(E2E_EXECUTOR)/' | sed 's/"true"/"false"/' > dist/no-db.yaml

dist/mysql.yaml: $(MANIFESTS) $(E2E_MANIFESTS) $(VERSION_FILE)
	kustomize build --load_restrictor=none test/e2e/manifests/mysql | sed 's/:$(MANIFESTS_VERSION)/:$(VERSION)/' | sed 's/pns/$(E2E_EXECUTOR)/' > dist/mysql.yaml

.PHONY: install
install: dist/postgres.yaml dist/mysql.yaml dist/no-db.yaml
ifeq ($(K3D),true)
	k3d start
endif
	# Install quick-start
	kubectl apply -f test/e2e/manifests/argo-ns.yaml
ifeq ($(DB),postgres)
	kubectl -n argo apply -f dist/postgres.yaml
else
ifeq ($(DB),mysql)
	kubectl -n argo apply -f dist/mysql.yaml
else
	kubectl -n argo apply -f dist/no-db.yaml
endif
endif

.PHONY: test-images
test-images: dist/cowsay-v1 dist/python-alpine3.6

dist/cowsay-v1:
	docker build -t cowsay:v1 test/e2e/images/cowsay
ifeq ($(K3D),true)
	k3d import-images cowsay:v1
endif
	touch dist/cowsay-v1

dist/python-alpine3.6:
	docker pull python:alpine3.6
	touch dist/python-alpine3.6

.PHONY: stop
stop:
	killall argo workflow-controller pf.sh kubectl || true

.PHONY: start-aux
start-aux:
	kubectl config set-context --current --namespace=argo
	kubectl -n argo wait --for=condition=Ready pod --all -l app --timeout 2m
	./hack/port-forward.sh
	# Check minio, postgres and mysql are in hosts file
	grep '127.0.0.1 *minio' /etc/hosts
	grep '127.0.0.1 *postgres' /etc/hosts
	grep '127.0.0.1 *mysql' /etc/hosts
ifneq ($(findstring controller,$(COMPONENTS)),)
	ALWAYS_OFFLOAD_NODE_STATUS=true OFFLOAD_NODE_STATUS_TTL=30s WORKFLOW_GC_PERIOD=30s UPPERIO_DB_DEBUG=1 ARCHIVED_WORKFLOW_GC_PERIOD=30s ./dist/workflow-controller --executor-image argoproj/argoexec:$(VERSION) --namespaced --loglevel debug &
endif
ifneq ($(findstring argo-server,$(COMPONENTS)),)
	UPPERIO_DB_DEBUG=1 ./dist/argo -v server --namespaced --auth-mode client --secure &
endif

.PHONY: start
start: status stop install controller cli executor-image start-aux wait env

.PHONY: wait
wait:
ifneq ($(findstring controller,$(COMPONENTS)),)
	# Wait for workflow controller
	until lsof -i :9090 > /dev/null ; do sleep 10s ; done
endif
ifneq ($(findstring argo-server,$(COMPONENTS)),)
	# Wait for Argo Server
	until lsof -i :2746 > /dev/null ; do sleep 10s ; done
endif

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
	kubectl -n argo logs -f -l app --max-log-requests 10 --tail 100

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
	go test -timeout 4m -v -count 1 -parallel 10 -run CronSuite ./test/e2e 2>&1 | tee test-results/test.out

.PHONY: smoke
smoke: test-images
	# Run smoke tests
	@mkdir -p test-results
	go test -timeout 1m -v -count 1 -p 1 -run SmokeSuite ./test/e2e 2>&1 | tee test-results/test.out

.PHONY: test-api
test-api: test-images
	# Run API tests
	go test -timeout 1m -v -count 1 -p 1 -run ArgoServerSuite ./test/e2e

.PHONY: test-cli
test-cli: test-images cli
	# Run CLI tests
	go test -timeout 2m -v -count 1 -p 1 -run CLISuite ./test/e2e
	go test -timeout 2m -v -count 1 -p 1 -run CLIWithServerSuite ./test/e2e

# clean

.PHONY: clean
clean:
	# Delete build files
	rm -Rf dist/* ui/dist

# swagger

$(HOME)/go/bin/swagger:
	$(call backup_go_mod)
	go get github.com/go-swagger/go-swagger/cmd/swagger@v0.23.0
	$(call restore_go_mod)

.PHONY: swagger
swagger: api/openapi-spec/swagger.json

pkg/apis/workflow/v1alpha1/openapi_generated.go:
	$(call backup_go_mod)
	go install k8s.io/kube-openapi/cmd/openapi-gen
	openapi-gen \
	  --go-header-file ./hack/custom-boilerplate.go.txt \
	  --input-dirs github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
	  --output-package github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
	  --report-filename pkg/apis/api-rules/violation_exceptions.list
	$(call restore_go_mod)

pkg/apiclient/_.secondary.swagger.json: hack/secondaryswaggergen.go pkg/apis/workflow/v1alpha1/openapi_generated.go
	go run ./hack secondaryswaggergen

api/openapi-spec/swagger.json: $(HOME)/go/bin/swagger pkg/apiclient/_.secondary.swagger.json $(SWAGGER_FILES) $(MANIFESTS_VERSION_FILE) hack/swaggify.sh
	swagger mixin -c 680 $(SWAGGER_FILES) | sed 's/VERSION/$(MANIFESTS_VERSION)/' | ./hack/swaggify.sh > api/openapi-spec/swagger.json

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
prepare-release: clean codegen manifests
	# Commit if any changes
	git diff --quiet || git commit -am "Update manifests to $(VERSION)"
	git tag $(VERSION)

.PHONY: publish-release
publish-release: build
	# Push images to Docker Hub
	docker push $(IMAGE_NAMESPACE)/argocli:$(VERSION)
	docker push $(IMAGE_NAMESPACE)/argoexec:$(VERSION)
	docker push $(IMAGE_NAMESPACE)/workflow-controller:$(VERSION)
	git push
	git push $(GIT_REMOTE) $(VERSION)
endif
