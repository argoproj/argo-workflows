export SHELL:=/bin/bash
export SHELLOPTS:=$(if $(SHELLOPTS),$(SHELLOPTS):)pipefail:errexit

# https://stackoverflow.com/questions/4122831/disable-make-builtin-rules-and-variables-from-inside-the-make-file
MAKEFLAGS += --no-builtin-rules
.SUFFIXES:

OUTPUT_IMAGE_OS ?= linux
OUTPUT_IMAGE_ARCH ?= amd64

BUILD_DATE             = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT             = $(shell git rev-parse HEAD)
GIT_REMOTE             = origin
GIT_BRANCH             = $(shell git rev-parse --symbolic-full-name --verify --quiet --abbrev-ref HEAD)
GIT_TAG                = $(shell git describe --always --tags --abbrev=0 || echo untagged)
GIT_TREE_STATE         = $(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)

export DOCKER_BUILDKIT = 1

# Use a different Dockerfile, e.g. for building for Windows or dev images.
DOCKERFILE            := Dockerfile

# docker image publishing options
IMAGE_NAMESPACE       ?= argoproj
# The name of the namespace where Kubernetes resources/RBAC will be installed
KUBE_NAMESPACE        ?= argo

VERSION               := latest

# VERSION is the version to be used for files in manifests and should always be latest uunlesswe are releasing
# we assume HEAD means you are on a tag
ifeq ($(findstring release,$(GIT_BRANCH)),release)
VERSION               := $(GIT_TAG)
endif

# version change, so does the file location
CLI_IMAGE_FILE         := dist/cli-image.marker

# perform static compilation
STATIC_BUILD          ?= true
STATIC_FILES          ?= true
GOTEST                ?= go test
# whether or not to start the Argo Service in TLS mode
SECURE                := false
# Which mode to run in:
# * `local` run the argo-server as single replicas on the local machine (default)
# * `kubernetes` run the argo-server on the Kubernetes cluster
RUN_MODE              := local
K3D                   := $(shell if [[ "`which kubectl`" != '' ]] && [[ "`kubectl config current-context`" == "k3d-"* ]]; then echo true; else echo false; fi)
LOG_LEVEL             := debug
UPPERIO_DB_DEBUG      := 0
NAMESPACED            := true

override LDFLAGS += \
  -X github.com/argoproj/argo-server.version=$(VERSION) \
  -X github.com/argoproj/argo-server.buildDate=${BUILD_DATE} \
  -X github.com/argoproj/argo-server.gitCommit=${GIT_COMMIT} \
  -X github.com/argoproj/argo-server.gitTreeState=${GIT_TREE_STATE}

ifeq ($(STATIC_BUILD), true)
override LDFLAGS += -extldflags "-static"
endif

ifneq ($(GIT_TAG),)
override LDFLAGS += -X github.com/argoproj/argo-server.gitTag=${GIT_TAG}
endif

CLI_PKGS         := $(shell echo cmd/argo                && go list -f '{{ join .Deps "\n" }}' ./cmd/argo/                | grep 'argoproj/argo-server/v3/' | cut -c 36-)
MANIFESTS        := $(shell find manifests -mindepth 2 -type f)
E2E_MANIFESTS    := $(shell find test/e2e/manifests -mindepth 2 -type f)
SWAGGER_FILES := pkg/apiclient/_.primary.swagger.json \
	pkg/apiclient/_.secondary.swagger.json \
	pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.swagger.json \
	pkg/apiclient/cronworkflow/cron-workflow.swagger.json \
	pkg/apiclient/event/event.swagger.json \
	pkg/apiclient/info/info.swagger.json \
	pkg/apiclient/workflow/workflow.swagger.json \
	pkg/apiclient/workflowarchive/workflow-archive.swagger.json \
	pkg/apiclient/workflowtemplate/workflow-template.swagger.json
PROTO_BINARIES := $(GOPATH)/bin/protoc-gen-gogo $(GOPATH)/bin/protoc-gen-gogofast $(GOPATH)/bin/goimports $(GOPATH)/bin/protoc-gen-grpc-gateway $(GOPATH)/bin/protoc-gen-swagger

# go_install,path
define go_install
	[ -e vendor ] || go mod vendor
	go install -mod=vendor ./vendor/$(1)
endef

# protoc,my.proto
define protoc
	# protoc $(1)
    [ -e vendor ] || go mod vendor
    protoc \
      -I /usr/local/include \
      -I . \
      -I ./vendor \
      -I ${GOPATH}/src \
      -I ${GOPATH}/pkg/mod/github.com/gogo/protobuf@v1.3.1/gogoproto \
      -I ${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.12.2/third_party/googleapis \
      --gogofast_out=plugins=grpc:${GOPATH}/src \
      --grpc-gateway_out=logtostderr=true:${GOPATH}/src \
      --swagger_out=logtostderr=true,fqn_for_swagger_name=true:. \
      $(1) 2>&1 | grep -v 'warning: Import .* is unused'
endef
# docker_build,image_name,binary_name,marker_file_name
define docker_build
	# If we're making a dev build, we build this locally (this will be faster due to existing Go build caches).
	docker build --progress plain -t $(IMAGE_NAMESPACE)/$(1):$(VERSION) --target $(1) -f $(DOCKERFILE) --build-arg IMAGE_OS=$(OUTPUT_IMAGE_OS) --build-arg IMAGE_ARCH=$(OUTPUT_IMAGE_ARCH) .
	if [ $(K3D) = true ]; then k3d image import $(IMAGE_NAMESPACE)/$(1):$(VERSION); fi
	touch $(3)
endef
define docker_pull
	docker pull $(1)
	if [ $(K3D) = true ]; then k3d image import $(1); fi
endef

ifndef $(GOPATH)
	GOPATH=$(shell go env GOPATH)
	export GOPATH
endif

.PHONY: build
build: clis images

.PHONY: images
images: cli-image

# cli

.PHONY: cli
cli: dist/argo argo-server.crt argo-server.key

ui/dist/app/index.html: $(shell find ui/src -type f && find ui -maxdepth 1 -type f)
	# Build UI
	@mkdir -p ui/dist/app
ifeq ($(STATIC_FILES),true)
	# `yarn install` is fast (~2s), so you can call it safely.
	JOBS=max yarn --cwd ui install
	# `yarn build` is slow, so we guard it with a up-to-date check.
	JOBS=max yarn --cwd ui build
else
	echo "Built without static files" > ui/dist/app/index.html
endif

$(GOPATH)/bin/staticfiles:
	go get bou.ke/staticfiles

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

dist/argo-%.gz: dist/argo-%
	gzip --force --keep dist/argo-$*

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
clis: dist/argo-linux-amd64.gz dist/argo-linux-arm64.gz dist/argo-linux-ppc64le.gz dist/argo-linux-s390x.gz dist/argo-darwin-amd64.gz dist/argo-windows-amd64.gz

# generation

.PHONY: codegen
codegen: \
	pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.swagger.json \
	pkg/apiclient/cronworkflow/cron-workflow.swagger.json \
	pkg/apiclient/event/event.swagger.json \
	pkg/apiclient/info/info.swagger.json \
	pkg/apiclient/workflow/workflow.swagger.json \
	pkg/apiclient/workflowarchive/workflow-archive.swagger.json \
	pkg/apiclient/workflowtemplate/workflow-template.swagger.json \
	manifests/install.yaml \
	api/openapi-spec/swagger.json \
	$(GOPATH)/bin/mockery
	# `go generate ./...` takes around 10s, so we only run on specific packages.
	go generate ./server/auth ./server/auth/sso
	rm -Rf vendor

$(GOPATH)/bin/mockery:
	./hack/recurl.sh dist/mockery.tar.gz https://github.com/vektra/mockery/releases/download/v1.1.1/mockery_1.1.1_$(shell uname -s)_$(shell uname -m).tar.gz
	tar zxvf dist/mockery.tar.gz mockery
	chmod +x mockery
	mkdir -p $(GOPATH)/bin
	mv mockery $(GOPATH)/bin/mockery
	mockery -version

$(GOPATH)/bin/go-to-protobuf:
	$(call go_install,k8s.io/code-generator/cmd/go-to-protobuf)

$(GOPATH)/bin/protoc-gen-gogo:
	$(call go_install,github.com/gogo/protobuf/protoc-gen-gogo)

$(GOPATH)/bin/protoc-gen-gogofast:
	$(call go_install,github.com/gogo/protobuf/protoc-gen-gogofast)

$(GOPATH)/bin/protoc-gen-grpc-gateway:
	$(call go_install,github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway)

$(GOPATH)/bin/protoc-gen-swagger:
	$(call go_install,github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger)

$(GOPATH)/bin/swagger:
	$(call go_install,github.com/go-swagger/go-swagger/cmd/swagger)

$(GOPATH)/bin/goimports:
	go get golang.org/x/tools/cmd/goimports@v0.0.0-20200630154851-b2d8b0336632


# this target will also create a .pb.go and a .pb.gw.go file, but in Make 3 we cannot use _grouped target_, instead we must choose
# on file to represent all of them
pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.proto
	$(call protoc,pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.proto)

pkg/apiclient/cronworkflow/cron-workflow.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/cronworkflow/cron-workflow.proto
	$(call protoc,pkg/apiclient/cronworkflow/cron-workflow.proto)

pkg/apiclient/event/event.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/event/event.proto
	$(call protoc,pkg/apiclient/event/event.proto)

pkg/apiclient/info/info.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/info/info.proto
	$(call protoc,pkg/apiclient/info/info.proto)

pkg/apiclient/workflow/workflow.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/workflow/workflow.proto
	$(call protoc,pkg/apiclient/workflow/workflow.proto)

pkg/apiclient/workflowarchive/workflow-archive.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/workflowarchive/workflow-archive.proto
	$(call protoc,pkg/apiclient/workflowarchive/workflow-archive.proto)

pkg/apiclient/workflowtemplate/workflow-template.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/workflowtemplate/workflow-template.proto
	$(call protoc,pkg/apiclient/workflowtemplate/workflow-template.proto)

/usr/local/bin/kustomize:
	mkdir -p dist
	./hack/recurl.sh dist/install_kustomize.sh https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh
	chmod +x ./dist/install_kustomize.sh
	./dist/install_kustomize.sh 3.8.8
	sudo mv kustomize /usr/local/bin/
	kustomize version

# generates several installation files
manifests/install.yaml: $(CRDS) /usr/local/bin/kustomize
	./hack/update-image-tags.sh manifests/base $(VERSION)
	kustomize build --load_restrictor=none manifests/cluster-install | ./hack/auto-gen-msg.sh > manifests/install.yaml
	kustomize build --load_restrictor=none manifests/namespace-install | ./hack/auto-gen-msg.sh > manifests/namespace-install.yaml
	kustomize build --load_restrictor=none manifests/quick-start/minimal | ./hack/auto-gen-msg.sh > manifests/quick-start-minimal.yaml

# lint/test/etc

$(GOPATH)/bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin v1.27.0

.PHONY: lint
lint: server/static/files.go $(GOPATH)/bin/golangci-lint
	rm -Rf vendor
	# Tidy Go modules
	go mod tidy
	# Lint Go files
	golangci-lint run --fix --verbose --concurrency 4 --timeout 5m
	# Lint UI files
ifeq ($(STATIC_FILES),true)
	yarn --cwd ui lint
endif

# for local we have a faster target that prints to stdout, does not use json, and can cache because it has no coverage
.PHONY: test
test: server/static/files.go
	env KUBECONFIG=/dev/null $(GOTEST) ./...

.PHONY: install
install: $(MANIFESTS) $(E2E_MANIFESTS) /usr/local/bin/kustomize
	kustomize build --load_restrictor=none test/e2e/manifests/minimal | sed 's/:latest/:$(VERSION)/' | kubectl -n $(KUBE_NAMESPACE) apply --force -f-

.PHONY: test-images
test-images:
	$(call docker_pull,argoproj/argosay:v2)

.PHONY: stop
stop:
	killall argo kubectl || true

$(GOPATH)/bin/goreman:
	go get github.com/mattn/goreman

.PHONY: start
start: stop install $(GOPATH)/bin/goreman
	kubectl config set-context --current --namespace=$(KUBE_NAMESPACE)
ifeq ($(RUN_MODE),kubernetes)
	$(MAKE) cli-image
	kubectl -n $(KUBE_NAMESPACE) scale deploy/argo-server --replicas 1
	kubectl -n $(KUBE_NAMESPACE) wait --for=condition=Ready pod -l app=argo-server
endif
	kubectl -n $(KUBE_NAMESPACE) wait --for=condition=Ready pod -l app=workflow-controller
	kubectl -n $(KUBE_NAMESPACE) wait --for=condition=Ready pod -l app=dex
	kubectl -n $(KUBE_NAMESPACE) wait --for=condition=Ready pod -l app=mysql
	./hack/port-forward.sh
	# Check dex, minio and mysql are in hosts file
	grep '127.0.0.1[[:blank:]]*dex' /etc/hosts
	grep '127.0.0.1[[:blank:]]*minio' /etc/hosts
	grep '127.0.0.1[[:blank:]]*mysql' /etc/hosts
ifeq ($(RUN_MODE),local)
	env SECURE=$(SECURE) LOG_LEVEL=$(LOG_LEVEL) UPPERIO_DB_DEBUG=$(UPPERIO_DB_DEBUG) VERSION=$(VERSION) NAMESPACED=$(NAMESPACED) NAMESPACE=$(KUBE_NAMESPACE) $(GOPATH)/bin/goreman -set-ports=false -logtime=false start
endif

.PHONY: wait
wait:
	# Wait for Argo Server
	until lsof -i :2746 > /dev/null ; do sleep 10s ; done

.PHONY: test-cli
test-cli: dist/argo
	$(GOTEST) -timeout 15m -count 1 --tags cli -p 1 --short ./test/e2e

# clean

.PHONY: clean
clean:
	go clean
	rm -Rf test-results node_modules vendor dist/* ui/dist

# swagger

dist/kubernetes.swagger.json:
	@mkdir -p dist
	./hack/recurl.sh dist/kubernetes.swagger.json https://raw.githubusercontent.com/kubernetes/kubernetes/v1.17.5/api/openapi-spec/swagger.json

pkg/apiclient/_.secondary.swagger.json: hack/swagger/secondaryswaggergen.go server/static/files.go dist/kubernetes.swagger.json
	# We have `hack/swagger` so that most hack script do not depend on the whole code base and are therefore slow.
	go run ./hack/swagger secondaryswaggergen

# we always ignore the conflicts, so lets automated figuring out how many there will be and just use that
dist/swagger-conflicts: $(GOPATH)/bin/swagger $(SWAGGER_FILES)
	swagger mixin $(SWAGGER_FILES) 2>&1 | grep -c skipping > dist/swagger-conflicts || true

dist/mixed.swagger.json: $(GOPATH)/bin/swagger $(SWAGGER_FILES) dist/swagger-conflicts
	swagger mixin -c $(shell cat dist/swagger-conflicts) $(SWAGGER_FILES) -o dist/mixed.swagger.json

dist/swaggifed.swagger.json: dist/mixed.swagger.json hack/swaggify.sh
	cat dist/mixed.swagger.json | sed 's/VERSION/$(VERSION)/' | ./hack/swaggify.sh > dist/swaggifed.swagger.json

dist/kubeified.swagger.json: dist/swaggifed.swagger.json dist/kubernetes.swagger.json
	go run ./hack/swagger kubeifyswagger dist/swaggifed.swagger.json dist/kubeified.swagger.json

api/openapi-spec/swagger.json: $(GOPATH)/bin/swagger dist/kubeified.swagger.json
	swagger flatten --with-flatten minimal --with-flatten remove-unused dist/kubeified.swagger.json -o api/openapi-spec/swagger.json
	swagger validate api/openapi-spec/swagger.json
	go test ./api/openapi-spec

# generates several other files
docs/cli/argo.md: $(CLI_PKGS) server/static/files.go hack/cli/main.go
	go run ./hack/cli

# pre-push

.PHONY: pre-commit
pre-commit: codegen lint test start

# release - targets only available on release branch
ifneq ($(findstring release,$(GIT_BRANCH)),)

.PHONY: prepare-release
prepare-release: check-version-warning clean codegen manifests
	# Commit if any changes
	git diff --quiet || git commit -am "Update manifests to $(VERSION)"
    # use "annotated" tag, rather than "lightweight", so in future we can distingush from "stable"
	git tag -a $(VERSION) -m $(VERSION)

.PHONY: publish-release
publish-release: check-version-warning clis
	git push
	git push $(GIT_REMOTE) $(VERSION)
endif

.PHONY: check-version-warning
check-version-warning:
	@if [[ "$(VERSION)" =~ ^[0-9]+\.[0-9]+\.[0-9]+.*$  ]]; then echo -n "It looks like you're trying to use a SemVer version, but have not prepended it with a "v" (such as "v$(VERSION)"). The "v" is required for our releases. Do you wish to continue anyway? [y/N]" && read ans && [ $${ans:-N} = y ]; fi

.PHONY: parse-examples
parse-examples:
	go run -tags fields ./hack parseexamples
