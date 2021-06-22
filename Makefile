export SHELL:=/bin/bash
export SHELLOPTS:=$(if $(SHELLOPTS),$(SHELLOPTS):)pipefail:errexit

# https://stackoverflow.com/questions/4122831/disable-make-builtin-rules-and-variables-from-inside-the-make-file
MAKEFLAGS += --no-builtin-rules
.SUFFIXES:

BUILD_DATE            := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT            := $(shell git rev-parse HEAD)
GIT_REMOTE            := origin
GIT_BRANCH            := $(shell git rev-parse --symbolic-full-name --verify --quiet --abbrev-ref HEAD)
GIT_TAG               := $(shell git describe --exact-match --tags --abbrev=0  2> /dev/null || echo untagged)
GIT_TREE_STATE        := $(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)
RELEASE_TAG           := $(shell if [[ "$(GIT_TAG)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+.*$$ ]]; then echo "true"; else echo "false"; fi)
DEV_BRANCH            := $(shell [ $(GIT_BRANCH) = master ] || [ `echo $(GIT_BRANCH) | cut -c -8` = release- ] || [ `echo $(GIT_BRANCH) | cut -c -4` = dev- ] || [ $(RELEASE_TAG) = true ] && echo false || echo true)

GREP_LOGS             := ""

# docker image publishing options
IMAGE_NAMESPACE       ?= argoproj
DEV_IMAGE             ?= $(shell [ `uname -s` = Darwin ] && echo true || echo false)

# The name of the namespace where Kubernetes resources/RBAC will be installed
KUBE_NAMESPACE        ?= argo

VERSION               := latest
DOCKER_PUSH           := false

# VERSION is the version to be used for files in manifests and should always be latest unless we are releasing
# we assume HEAD means you are on a tag
ifeq ($(RELEASE_TAG),true)
VERSION               := $(GIT_TAG)
endif

# should we build the static files?
ifneq (,$(filter $(MAKECMDGOALS),codegen lint test docs start))
STATIC_FILES          := false
else
STATIC_FILES          ?= $(shell [ $(DEV_BRANCH) = true ] && echo false || echo true)
endif

UI                    ?= true
GOTEST                ?= go test -v
PROFILE               ?= minimal
# by keeping this short we speed up the tests
DEFAULT_REQUEUE_TIME  ?= 100ms
# whether or not to start the Argo Service in TLS mode
SECURE                := false
AUTH_MODE             := hybrid
ifeq ($(PROFILE),sso)
AUTH_MODE             := sso
endif

# Which mode to run in:
# * `local` run the workflow–controller and argo-server as single replicas on the local machine (default)
# * `kubernetes` run the workflow-controller and argo-server on the Kubernetes cluster
RUN_MODE              := local
KUBECTX               := $(shell [[ "`which kubectl`" != '' ]] && kubectl config current-context || echo none)
DOCKER_DESKTOP        := $(shell [[ "$(KUBECTX)" == "docker-desktop" ]] && echo true || echo false)
K3D                   := $(shell [[ "$(KUBECTX)" == "k3d-"* ]] && echo true || echo false)
LOG_LEVEL             := debug
UPPERIO_DB_DEBUG      := 0
NAMESPACED            := true
ifeq ($(PROFILE),prometheus)
RUN_MODE              := kubernetes
endif
ifeq ($(PROFILE),stress)
RUN_MODE              := kubernetes
endif

ALWAYS_OFFLOAD_NODE_STATUS := false

$(info GIT_COMMIT=$(GIT_COMMIT) GIT_BRANCH=$(GIT_BRANCH) GIT_TAG=$(GIT_TAG) GIT_TREE_STATE=$(GIT_TREE_STATE) RELEASE_TAG=$(RELEASE_TAG) DEV_BRANCH=$(DEV_BRANCH) VERSION=$(VERSION))
$(info KUBECTX=$(KUBECTX) DOCKER_DESKTOP=$(DOCKER_DESKTOP) K3D=$(K3D) DOCKER_PUSH=$(DOCKER_PUSH))
$(info RUN_MODE=$(RUN_MODE) PROFILE=$(PROFILE) AUTH_MODE=$(AUTH_MODE) SECURE=$(SECURE) STATIC_FILES=$(STATIC_FILES) ALWAYS_OFFLOAD_NODE_STATUS=$(ALWAYS_OFFLOAD_NODE_STATUS) UPPERIO_DB_DEBUG=$(UPPERIO_DB_DEBUG) LOG_LEVEL=$(LOG_LEVEL) NAMESPACED=$(NAMESPACED))

override LDFLAGS += \
  -X github.com/argoproj/argo-workflows/v3.version=$(VERSION) \
  -X github.com/argoproj/argo-workflows/v3.buildDate=${BUILD_DATE} \
  -X github.com/argoproj/argo-workflows/v3.gitCommit=${GIT_COMMIT} \
  -X github.com/argoproj/argo-workflows/v3.gitTreeState=${GIT_TREE_STATE}

ifneq ($(GIT_TAG),)
override LDFLAGS += -X github.com/argoproj/argo-workflows/v3.gitTag=${GIT_TAG}
endif

ARGOEXEC_PKGS    := $(shell echo cmd/argoexec            && go list -f '{{ join .Deps "\n" }}' ./cmd/argoexec/            | grep 'argoproj/argo-workflows/v3/' | cut -c 39-)
CLI_PKGS         := $(shell echo cmd/argo                && go list -f '{{ join .Deps "\n" }}' ./cmd/argo/                | grep 'argoproj/argo-workflows/v3/' | cut -c 39-)
CONTROLLER_PKGS  := $(shell echo cmd/workflow-controller && go list -f '{{ join .Deps "\n" }}' ./cmd/workflow-controller/ | grep 'argoproj/argo-workflows/v3/' | cut -c 39-)
E2E_EXECUTOR ?= pns
TYPES := $(shell find pkg/apis/workflow/v1alpha1 -type f -name '*.go' -not -name openapi_generated.go -not -name '*generated*' -not -name '*test.go')
CRDS := $(shell find manifests/base/crds -type f -name 'argoproj.io_*.yaml')
SWAGGER_FILES := pkg/apiclient/_.primary.swagger.json \
	pkg/apiclient/_.secondary.swagger.json \
	pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.swagger.json \
	pkg/apiclient/cronworkflow/cron-workflow.swagger.json \
	pkg/apiclient/event/event.swagger.json \
	pkg/apiclient/eventsource/eventsource.swagger.json \
	pkg/apiclient/info/info.swagger.json \
	pkg/apiclient/sensor/sensor.swagger.json \
	pkg/apiclient/workflow/workflow.swagger.json \
	pkg/apiclient/workflowarchive/workflow-archive.swagger.json \
	pkg/apiclient/workflowtemplate/workflow-template.swagger.json
PROTO_BINARIES := $(GOPATH)/bin/protoc-gen-gogo $(GOPATH)/bin/protoc-gen-gogofast $(GOPATH)/bin/goimports $(GOPATH)/bin/protoc-gen-grpc-gateway $(GOPATH)/bin/protoc-gen-swagger

# go_install,path
define go_install
	go mod vendor
	go install -mod=vendor ./vendor/$(1)
endef

# protoc,my.proto
define protoc
	# protoc $(1)
    [ -e ./vendor ] || go mod vendor
    protoc \
      -I /usr/local/include \
      -I $(CURDIR) \
      -I $(CURDIR)/vendor \
      -I $(GOPATH)/src \
      -I $(GOPATH)/pkg/mod/github.com/gogo/protobuf@v1.3.1/gogoproto \
      -I $(GOPATH)/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.16.0/third_party/googleapis \
      --gogofast_out=plugins=grpc:$(GOPATH)/src \
      --grpc-gateway_out=logtostderr=true:$(GOPATH)/src \
      --swagger_out=logtostderr=true,fqn_for_swagger_name=true:. \
      $(1)
     perl -i -pe 's|argoproj/argo-workflows/|argoproj/argo-workflows/v3/|g' `echo "$(1)" | sed 's/proto/pb.go/g'`

endef

ifndef $(GOPATH)
	GOPATH=$(shell go env GOPATH)
	export GOPATH
endif

.PHONY: build
build: clis images

.PHONY: images
images: argocli-image argoexec-image workflow-controller-image

# cli

.PHONY: cli
cli: dist/argo

ui/dist/app/index.html: $(shell find ui/src -type f && find ui -maxdepth 1 -type f)
	# `yarn install` is fast (~2s), so you can call it safely.
	JOBS=max yarn --cwd ui install
	# `yarn build` is slow, so we guard it with a up-to-date check.
	JOBS=max yarn --cwd ui build

$(GOPATH)/bin/staticfiles:
	cd `mktemp -d` && go get bou.ke/staticfiles

ifeq ($(STATIC_FILES),true)
server/static/files.go: $(GOPATH)/bin/staticfiles ui/dist/app/index.html
	# Pack UI into a Go file
	$(GOPATH)/bin/staticfiles -o server/static/files.go ui/dist/app
else
server/static/files.go:
	# Building without static files
	cp ./server/static/files.go.stub ./server/static/files.go
endif

dist/argo-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/argo-darwin-amd64: GOARGS = GOOS=darwin GOARCH=amd64
dist/argo-windows-amd64: GOARGS = GOOS=windows GOARCH=amd64
dist/argo-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64
dist/argo-linux-ppc64le: GOARGS = GOOS=linux GOARCH=ppc64le
dist/argo-linux-s390x: GOARGS = GOOS=linux GOARCH=s390x

dist/argo-%.gz: dist/argo-%
	gzip --force --keep dist/argo-$*

dist/argo-%: server/static/files.go argo-server.crt argo-server.key $(CLI_PKGS) go.sum
	CGO_ENABLED=0 $(GOARGS) go build -v -ldflags '${LDFLAGS} -extldflags -static' -o $@ ./cmd/argo

dist/argo: server/static/files.go argo-server.crt argo-server.key $(CLI_PKGS) go.sum
ifeq ($(shell uname -s),Darwin)
	# if local, then build fast: use CGO and dynamic-linking
	go build -v -ldflags '${LDFLAGS}' -o $@ ./cmd/argo
else
	CGO_ENABLED=0 go build -v -ldflags '${LDFLAGS} -extldflags -static' -o $@ ./cmd/argo
endif

argo-server.crt: argo-server.key

argo-server.key:
	openssl req -x509 -newkey rsa:4096 -keyout argo-server.key -out argo-server.crt -days 365 -nodes -subj /CN=localhost/O=ArgoProj

argocli-image:

.PHONY: clis
clis: dist/argo-linux-amd64.gz dist/argo-linux-arm64.gz dist/argo-linux-ppc64le.gz dist/argo-linux-s390x.gz dist/argo-darwin-amd64.gz dist/argo-windows-amd64.gz

# controller

.PHONY: controller
controller: dist/workflow-controller

dist/workflow-controller: $(CONTROLLER_PKGS) go.sum
ifeq ($(shell uname -s),Darwin)
	# if local, then build fast: use CGO and dynamic-linking
	go build -v -ldflags '${LDFLAGS}' -o $@ ./cmd/workflow-controller
else
	CGO_ENABLED=0 go build -v -ldflags '${LDFLAGS} -extldflags -static' -o $@ ./cmd/workflow-controller
endif

workflow-controller-image:

# argoexec

dist/argoexec: $(ARGOEXEC_PKGS) go.sum
ifeq ($(shell uname -s),Darwin)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS} -extldflags -static' -o $@ ./cmd/argoexec
else
	CGO_ENABLED=0 go build -v -ldflags '${LDFLAGS} -extldflags -static' -o $@ ./cmd/argoexec
endif

argoexec-image:

%-image:
	[ ! -e dist/$* ] || mv dist/$* .
	docker buildx build -t $(IMAGE_NAMESPACE)/$*:$(VERSION) --target $* --output=type=docker .
	[ ! -e $* ] || mv $* dist/
	docker run --rm -t $(IMAGE_NAMESPACE)/$*:$(VERSION) version
	if [ $(K3D) = true ]; then k3d image import $(IMAGE_NAMESPACE)/$*:$(VERSION); fi
	if [ $(DOCKER_PUSH) = true ] && [ $(IMAGE_NAMESPACE) != argoproj ] ; then docker push $(IMAGE_NAMESPACE)/$*:$(VERSION) ; fi

scan-images: scan-workflow-controller scan-argoexec scan-argocli

scan-%:
	docker scan --severity=high $(IMAGE_NAMESPACE)/$*:$(VERSION)

# generation

.PHONY: codegen
codegen: types swagger docs manifests

.PHONY: types
types: pkg/apis/workflow/v1alpha1/generated.proto pkg/apis/workflow/v1alpha1/openapi_generated.go pkg/apis/workflow/v1alpha1/zz_generated.deepcopy.go

.PHONY: swagger
swagger: \
	pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.swagger.json \
	pkg/apiclient/cronworkflow/cron-workflow.swagger.json \
	pkg/apiclient/event/event.swagger.json \
	pkg/apiclient/eventsource/eventsource.swagger.json \
	pkg/apiclient/info/info.swagger.json \
	pkg/apiclient/sensor/sensor.swagger.json \
	pkg/apiclient/workflow/workflow.swagger.json \
	pkg/apiclient/workflowarchive/workflow-archive.swagger.json \
	pkg/apiclient/workflowtemplate/workflow-template.swagger.json \
	manifests/base/crds/full/argoproj.io_workflows.yaml \
	manifests \
	api/openapi-spec/swagger.json \
	api/jsonschema/schema.json

.PHONY: docs
docs: \
	docs/fields.md \
	docs/cli/argo.md \
	$(GOPATH)/bin/mockery
	rm -Rf vendor v3
	go mod tidy
	# `go generate ./...` takes around 10s, so we only run on specific packages.
	go generate ./persist/sqldb ./pkg/apiclient/workflow ./server/auth ./server/auth/sso ./workflow/executor
	./hack/check-env-doc.sh

$(GOPATH)/bin/mockery:
	./hack/recurl.sh dist/mockery.tar.gz https://github.com/vektra/mockery/releases/download/v1.1.1/mockery_1.1.1_$(shell uname -s)_$(shell uname -m).tar.gz
	tar zxvf dist/mockery.tar.gz mockery
	chmod +x mockery
	mkdir -p $(GOPATH)/bin
	mv mockery $(GOPATH)/bin/mockery
	mockery -version

$(GOPATH)/bin/controller-gen:
	$(call go_install,sigs.k8s.io/controller-tools/cmd/controller-gen)

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

$(GOPATH)/bin/openapi-gen:
	$(call go_install,k8s.io/kube-openapi/cmd/openapi-gen)

$(GOPATH)/bin/swagger:
	$(call go_install,github.com/go-swagger/go-swagger/cmd/swagger)

$(GOPATH)/bin/goimports:
	$(call go_install,golang.org/x/tools/cmd/goimports)

pkg/apis/workflow/v1alpha1/generated.proto: $(GOPATH)/bin/go-to-protobuf $(PROTO_BINARIES) $(TYPES)
	[ -e ./vendor ] || go mod vendor
	[ -e ./v3 ] || ln -s . v3
	$(GOPATH)/bin/go-to-protobuf \
		--go-header-file=./hack/custom-boilerplate.go.txt \
		--packages=github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1 \
		--apimachinery-packages=+k8s.io/apimachinery/pkg/util/intstr,+k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/runtime/schema,+k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1,k8s.io/api/policy/v1beta1 \
		--proto-import $(CURDIR)/vendor
	touch pkg/apis/workflow/v1alpha1/generated.proto
	[ -e ./v3 ] && rm -rf v3

# this target will also create a .pb.go and a .pb.gw.go file, but in Make 3 we cannot use _grouped target_, instead we must choose
# on file to represent all of them
pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.proto
	$(call protoc,pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.proto)

pkg/apiclient/cronworkflow/cron-workflow.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/cronworkflow/cron-workflow.proto
	$(call protoc,pkg/apiclient/cronworkflow/cron-workflow.proto)

pkg/apiclient/event/event.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/event/event.proto
	$(call protoc,pkg/apiclient/event/event.proto)

pkg/apiclient/eventsource/eventsource.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/eventsource/eventsource.proto
	$(call protoc,pkg/apiclient/eventsource/eventsource.proto)

pkg/apiclient/info/info.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/info/info.proto
	$(call protoc,pkg/apiclient/info/info.proto)

pkg/apiclient/sensor/sensor.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/sensor/sensor.proto
	$(call protoc,pkg/apiclient/sensor/sensor.proto)

pkg/apiclient/workflow/workflow.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/workflow/workflow.proto
	$(call protoc,pkg/apiclient/workflow/workflow.proto)

pkg/apiclient/workflowarchive/workflow-archive.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/workflowarchive/workflow-archive.proto
	$(call protoc,pkg/apiclient/workflowarchive/workflow-archive.proto)

pkg/apiclient/workflowtemplate/workflow-template.swagger.json: $(PROTO_BINARIES) $(TYPES) pkg/apiclient/workflowtemplate/workflow-template.proto
	$(call protoc,pkg/apiclient/workflowtemplate/workflow-template.proto)

# generate other files for other CRDs
manifests/base/crds/full/argoproj.io_workflows.yaml: $(GOPATH)/bin/controller-gen $(TYPES) ./hack/crdgen.sh ./hack/crds.go
	./hack/crdgen.sh

dist/kustomize:
	mkdir -p dist
	rm -f dist/kustomize
	./hack/recurl.sh dist/install_kustomize.sh https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh
	chmod +x ./dist/install_kustomize.sh
	cd dist && ./install_kustomize.sh 3.8.8
	dist/kustomize version

manifests: \
	manifests/install.yaml \
	manifests/namespace-install.yaml \
	manifests/quick-start-minimal.yaml \
	manifests/quick-start-mysql.yaml \
	manifests/quick-start-postgres.yaml \
	dist/manifests/install.yaml \
	dist/manifests/namespace-install.yaml \
	dist/manifests/quick-start-minimal.yaml \
	dist/manifests/quick-start-mysql.yaml \
	dist/manifests/quick-start-postgres.yaml

manifests/install.yaml: dist/kustomize /dev/null
	dist/kustomize build --load_restrictor=none manifests/cluster-install | ./hack/auto-gen-msg.sh > manifests/install.yaml
manifests/namespace-install.yaml: dist/kustomize /dev/null
	dist/kustomize build --load_restrictor=none manifests/namespace-install | ./hack/auto-gen-msg.sh > manifests/namespace-install.yaml
manifests/quick-start-minimal.yaml: dist/kustomize /dev/null
	dist/kustomize build --load_restrictor=none manifests/quick-start/minimal | ./hack/auto-gen-msg.sh > manifests/quick-start-minimal.yaml
manifests/quick-start-mysql.yaml: dist/kustomize /dev/null
	dist/kustomize build --load_restrictor=none manifests/quick-start/mysql | ./hack/auto-gen-msg.sh > manifests/quick-start-mysql.yaml
manifests/quick-start-postgres.yaml: dist/kustomize /dev/null
	dist/kustomize build --load_restrictor=none manifests/quick-start/postgres | ./hack/auto-gen-msg.sh > manifests/quick-start-postgres.yaml

dist/manifests/%: manifests/%
	@mkdir -p dist/manifests
	sed 's/:latest/:$(VERSION)/' manifests/$* > $@

# lint/test/etc

$(GOPATH)/bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin v1.36.0

.PHONY: lint
lint: server/static/files.go $(GOPATH)/bin/golangci-lint
	rm -Rf v3 vendor
	# Tidy Go modules
	go mod tidy

	# Lint Go files
	$(GOPATH)/bin/golangci-lint run --fix --verbose

# for local we have a faster target that prints to stdout, does not use json, and can cache because it has no coverage
.PHONY: test
test: server/static/files.go dist/argosay
	env KUBECONFIG=/dev/null $(GOTEST) ./...

.PHONY: install
install: dist/kustomize
	kubectl get ns $(KUBE_NAMESPACE) || kubectl create ns $(KUBE_NAMESPACE)
	kubectl config set-context --current --namespace=$(KUBE_NAMESPACE)
	@echo "installing PROFILE=$(PROFILE), E2E_EXECUTOR=$(E2E_EXECUTOR)"
	dist/kustomize build --load_restrictor=none test/e2e/manifests/$(PROFILE) | sed 's/argoproj\//$(IMAGE_NAMESPACE)\//' | sed 's/containerRuntimeExecutor: docker/containerRuntimeExecutor: $(E2E_EXECUTOR)/' | kubectl -n $(KUBE_NAMESPACE) apply --prune -l app.kubernetes.io/part-of=argo -f -
ifeq ($(PROFILE),stress)
	kubectl -n $(KUBE_NAMESPACE) apply -f test/stress/massive-workflow.yaml
endif
ifeq ($(RUN_MODE),kubernetes)
	# scale to 2 replicas so we touch upon leader election
	kubectl -n $(KUBE_NAMESPACE) scale deploy/workflow-controller --replicas 2
	kubectl -n $(KUBE_NAMESPACE) scale deploy/argo-server --replicas 1
endif

# nuke is like "clean" but attempts to return you to a state as if it had never been installed at all
# this only available in "safe" local development
ifeq (true,$(filter true, $(DOCKER_DESKTOP) $(K3D)))
nuke:
	kubectl delete --ignore-not-found workflow,cronworkflow,clusterworkflowtemplate,workflowtemplate,workfloweventbinding --all || true
	kubectl delete --ignore-not-found ns argo
	kubectl delete --ignore-not-found -k manifests/base/crds/minimal
	git clean -fxd
	docker image rm argoproj/argoexec argoproj/argocli argoproj/workflow-controller || true
	docker system prune -f
endif

.PHONY: argosay
argosay:
	cd test/e2e/images/argosay/v2 && docker build . -t argoproj/argosay:v2
ifeq ($(K3D),true)
	k3d image import argoproj/argosay:v2
endif
ifeq ($(DOCKER_PUSH),true)
	docker push argoproj/argosay:v2
endif

dist/argosay:
	mkdir -p dist
	cp test/e2e/images/argosay/v2/argosay dist/

.PHONY: pull-images
pull-images:
	docker pull golang:1.15.7
	docker pull debian:10.7-slim
	docker pull mysql:8
	docker pull argoproj/argosay:v1
	docker pull argoproj/argosay:v2
	docker pull python:alpine3.6

$(GOPATH)/bin/goreman:
	cd `mktemp -d` && go get github.com/mattn/goreman

.PHONY: start
ifeq ($(RUN_MODE),local)
start: install controller cli $(GOPATH)/bin/goreman
else
start: install
endif
	@echo "starting STATIC_FILES=$(STATIC_FILES) (DEV_BRANCH=$(DEV_BRANCH), GIT_BRANCH=$(GIT_BRANCH)), AUTH_MODE=$(AUTH_MODE), RUN_MODE=$(RUN_MODE)"
	# Check dex, minio, postgres and mysql are in hosts file
ifeq ($(AUTH_MODE),sso)
	grep '127.0.0.1[[:blank:]]*dex' /etc/hosts
endif
	grep '127.0.0.1[[:blank:]]*minio' /etc/hosts
	grep '127.0.0.1[[:blank:]]*postgres' /etc/hosts
	grep '127.0.0.1[[:blank:]]*mysql' /etc/hosts
	./hack/port-forward.sh
ifeq ($(RUN_MODE),local)
	env DEFAULT_REQUEUE_TIME=$(DEFAULT_REQUEUE_TIME) SECURE=$(SECURE) ALWAYS_OFFLOAD_NODE_STATUS=$(ALWAYS_OFFLOAD_NODE_STATUS) LOG_LEVEL=$(LOG_LEVEL) UPPERIO_DB_DEBUG=$(UPPERIO_DB_DEBUG) IMAGE_NAMESPACE=$(IMAGE_NAMESPACE) VERSION=$(VERSION) AUTH_MODE=$(AUTH_MODE) NAMESPACED=$(NAMESPACED) NAMESPACE=$(KUBE_NAMESPACE) UI=$(UI) $(GOPATH)/bin/goreman -set-ports=false -logtime=false start $(shell if [ -z $GREP_LOGS ]; then echo; else echo "| grep \"$(GREP_LOGS)\""; fi)
endif

$(GOPATH)/bin/stern:
	./hack/recurl.sh $(GOPATH)/bin/stern https://github.com/wercker/stern/releases/download/1.11.0/stern_`uname -s|tr '[:upper:]' '[:lower:]'`_amd64

.PHONY: logs
logs: $(GOPATH)/bin/stern
	stern -l workflows.argoproj.io/workflow 2>&1

.PHONY: wait
wait:
	# Wait for workflow controller
	until lsof -i :9090 > /dev/null ; do sleep 10s ; done
	# Wait for Argo Server
	until lsof -i :2746 > /dev/null ; do sleep 10s ; done

.PHONY: postgres-cli
postgres-cli:
	kubectl exec -ti `kubectl get pod -l app=postgres -o name|cut -c 5-` -- psql -U postgres

.PHONY: mysql-cli
mysql-cli:
	kubectl exec -ti `kubectl get pod -l app=mysql -o name|cut -c 5-` -- mysql -u mysql -ppassword argo

start-e2e:
	$(MAKE) start PROFILE=mysql E2E_EXECUTOR=$(E2E_EXECUTOR) ALWAYS_OFFLOAD_NODE_STATUS=true AUTH_MODE=client

test-e2e: test-api test-cli test-cron test-executor test-functional

test-cli: ./dist/argo

test-%:
	$(GOTEST) -timeout 15m -count 1 --tags $* -parallel 10 ./test/e2e

.PHONY: test-examples
test-examples: ./dist/argo
	./hack/test-examples.sh

# clean

.PHONY: clean
clean:
	go clean
	rm -Rf test-results node_modules vendor v2 argoexec-linux-amd64 dist/* ui/dist

# swagger

pkg/apis/workflow/v1alpha1/openapi_generated.go: $(GOPATH)/bin/openapi-gen $(TYPES)
	[ -e ./v3 ] || ln -s . v3
	$(GOPATH)/bin/openapi-gen \
	  --go-header-file ./hack/custom-boilerplate.go.txt \
	  --input-dirs github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1 \
	  --output-package github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1 \
	  --report-filename pkg/apis/api-rules/violation_exceptions.list
	[ -e ./v3 ] && rm -rf v3


# generates many other files (listers, informers, client etc).
pkg/apis/workflow/v1alpha1/zz_generated.deepcopy.go: $(TYPES)
	[ -e ./vendor ] || go mod vendor
	[ -e ./v3 ] || ln -s . v3
	bash $(GOPATH)/pkg/mod/k8s.io/code-generator@v0.19.6/generate-groups.sh \
	    "deepcopy,client,informer,lister" \
	    github.com/argoproj/argo-workflows/v3/pkg/client github.com/argoproj/argo-workflows/v3/pkg/apis \
	    workflow:v1alpha1 \
	    --go-header-file ./hack/custom-boilerplate.go.txt
	[ -e ./v3 ] && rm -rf v3

dist/kubernetes.swagger.json:
	@mkdir -p dist
	./hack/recurl.sh dist/kubernetes.swagger.json https://raw.githubusercontent.com/kubernetes/kubernetes/v1.17.5/api/openapi-spec/swagger.json

pkg/apiclient/_.secondary.swagger.json: hack/swagger/secondaryswaggergen.go pkg/apis/workflow/v1alpha1/openapi_generated.go dist/kubernetes.swagger.json
	rm -Rf v3 vendor
	# We have `hack/swagger` so that most hack script do not depend on the whole code base and are therefore slow.
	go run ./hack/swagger secondaryswaggergen

# we always ignore the conflicts, so lets automated figuring out how many there will be and just use that
dist/swagger-conflicts: $(GOPATH)/bin/swagger $(SWAGGER_FILES)
	swagger mixin $(SWAGGER_FILES) 2>&1 | grep -c skipping > dist/swagger-conflicts || true

dist/mixed.swagger.json: $(GOPATH)/bin/swagger $(SWAGGER_FILES) dist/swagger-conflicts
	swagger mixin -c $(shell cat dist/swagger-conflicts) $(SWAGGER_FILES) -o dist/mixed.swagger.json

dist/swaggifed.swagger.json: dist/mixed.swagger.json hack/swaggify.sh
	cat dist/mixed.swagger.json | ./hack/swaggify.sh > dist/swaggifed.swagger.json

dist/kubeified.swagger.json: dist/swaggifed.swagger.json dist/kubernetes.swagger.json
	go run ./hack/swagger kubeifyswagger dist/swaggifed.swagger.json dist/kubeified.swagger.json

api/openapi-spec/swagger.json: $(GOPATH)/bin/swagger dist/kubeified.swagger.json
	swagger flatten --with-flatten minimal --with-flatten remove-unused dist/kubeified.swagger.json -o api/openapi-spec/swagger.json

api/jsonschema/schema.json: api/openapi-spec/swagger.json hack/jsonschema/main.go
	go run ./hack/jsonschema

go-diagrams/diagram.dot: ./hack/diagram/main.go
	rm -Rf go-diagrams
	go run ./hack/diagram

docs/assets/diagram.png: go-diagrams/diagram.dot
	cd go-diagrams && dot -Tpng diagram.dot -o ../docs/assets/diagram.png

docs/fields.md: api/openapi-spec/swagger.json $(shell find examples -type f) hack/docgen.go
	env ARGO_SECURE=false ARGO_INSECURE_SKIP_VERIFY=false ARGO_SERVER= ARGO_INSTANCEID= go run ./hack docgen

# generates several other files
docs/cli/argo.md: $(CLI_PKGS) go.sum server/static/files.go hack/cli/main.go
	go run ./hack/cli

.PHONY: validate-examples
validate-examples: api/jsonschema/schema.json
	cd examples && go test

# pre-push

.PHONY: pre-commit
pre-commit: codegen lint test start

ifeq ($(GIT_BRANCH),master)
LOG_OPTS := '-n10'
else
LOG_OPTS := 'origin/master..'
endif

release-notes: /dev/null
	version=$(VERSION) envsubst < hack/release-notes.md > release-notes

.PHONY: parse-examples
parse-examples:
	go run -tags fields ./hack parseexamples

.PHONY: checksums
checksums:
	for f in ./dist/argo-*.gz; do openssl dgst -sha256 "$$f" | awk ' { print $$2 }' > "$$f".sha256 ; done
