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
DEV_IMAGE             := true

# VERSION is the version to be used for files in manifests and should always be latest uunlesswe are releasing
# we assume HEAD means you are on a tag
ifeq ($(findstring release,$(GIT_BRANCH)),release)
VERSION               := $(GIT_TAG)
DEV_IMAGE             := false
endif

# If we are building dev images, then we want to use the Docker cache for speed.
ifeq ($(DEV_IMAGE),true)
DOCKERFILE            := Dockerfile.dev
endif

# version change, so does the file location
CLI_IMAGE_FILE         := dist/cli-image.marker
EXECUTOR_IMAGE_FILE    := dist/executor-image.marker
CONTROLLER_IMAGE_FILE  := dist/controller-image.marker

# perform static compilation
STATIC_BUILD          ?= true
STATIC_FILES          ?= true
GOTEST                ?= go test
PROFILE               ?= minimal
# by keeping this short we speed up the tests
DEFAULT_REQUEUE_TIME  ?= 2s
# whether or not to start the Argo Service in TLS mode
SECURE                := false
AUTH_MODE             := hybrid
ifeq ($(PROFILE),sso)
AUTH_MODE             := sso
endif
ifeq ($(STATIC_FILES),false)
AUTH_MODE             := client
endif
# Which mode to run in:
# * `local` run the workflow–controller and argo-server as single replicas on the local machine (default)
# * `kubernetes` run the workflow-controller and argo-server on the Kubernetes cluster
RUN_MODE              := local
K3D                   := $(shell if [[ "`which kubectl`" != '' ]] && [[ "`kubectl config current-context`" == "k3d-"* ]]; then echo true; else echo false; fi)
LOG_LEVEL             := debug
UPPERIO_DB_DEBUG      := 0
NAMESPACED            := true

ALWAYS_OFFLOAD_NODE_STATUS := false
ifeq ($(PROFILE),mysql)
ALWAYS_OFFLOAD_NODE_STATUS := true
endif
ifeq ($(PROFILE),postgres)
ALWAYS_OFFLOAD_NODE_STATUS := true
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
MANIFESTS        := $(shell find manifests -mindepth 2 -type f)
E2E_MANIFESTS    := $(shell find test/e2e/manifests -mindepth 2 -type f)
E2E_EXECUTOR ?= pns
TYPES := $(shell find pkg/apis/workflow/v1alpha1 -type f -name '*.go' -not -name openapi_generated.go -not -name '*generated*' -not -name '*test.go')
CRDS := $(shell find manifests/base/crds -type f -name 'argoproj.io_*.yaml')
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
	if [ $(DEV_IMAGE) = true ]; then $(MAKE) dist/$(2)-$(OUTPUT_IMAGE_OS)-$(OUTPUT_IMAGE_ARCH) && mv dist/$(2)-$(OUTPUT_IMAGE_OS)-$(OUTPUT_IMAGE_ARCH) $(2); fi
	docker build --progress plain -t $(IMAGE_NAMESPACE)/$(1):$(VERSION) --target $(1) -f $(DOCKERFILE) --build-arg IMAGE_OS=$(OUTPUT_IMAGE_OS) --build-arg IMAGE_ARCH=$(OUTPUT_IMAGE_ARCH) .
	if [ $(DEV_IMAGE) = true ]; then mv $(2) dist/$(2)-$(OUTPUT_IMAGE_OS)-$(OUTPUT_IMAGE_ARCH); fi
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
images: cli-image executor-image controller-image

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

.PHONY: controller
controller: dist/workflow-controller

dist/workflow-controller: GOARGS = GOOS= GOARCH=
dist/workflow-controller-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64
dist/workflow-controller-linux-arm64: GOARGS = GOOS=linux GOARCH=arm64
dist/workflow-controller-linux-ppc64le: GOARGS = GOOS=linux GOARCH=ppc64le
dist/workflow-controller-linux-s390x: GOARGS = GOOS=linux GOARCH=s390x

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
dist/argoexec-linux-ppc64le: GOARGS = GOOS=linux GOARCH=ppc64le
dist/argoexec-linux-s390x: GOARGS = GOOS=linux GOARCH=s390x

dist/argoexec-%: $(ARGOEXEC_PKGS)
	CGO_ENABLED=0 $(GOARGS) go build -v -i -ldflags '${LDFLAGS}' -o $@ ./cmd/argoexec

.PHONY: executor-image
executor-image: $(EXECUTOR_IMAGE_FILE)

$(EXECUTOR_IMAGE_FILE): $(ARGOEXEC_PKGS)
	# Create executor image
	$(call docker_build,argoexec,argoexec,$(EXECUTOR_IMAGE_FILE))

# generation

.PHONY: codegen
codegen: \
	pkg/apis/workflow/v1alpha1/generated.proto \
	pkg/apiclient/clusterworkflowtemplate/cluster-workflow-template.swagger.json \
	pkg/apiclient/cronworkflow/cron-workflow.swagger.json \
	pkg/apiclient/event/event.swagger.json \
	pkg/apiclient/info/info.swagger.json \
	pkg/apiclient/workflow/workflow.swagger.json \
	pkg/apiclient/workflowarchive/workflow-archive.swagger.json \
	pkg/apiclient/workflowtemplate/workflow-template.swagger.json \
	pkg/apis/workflow/v1alpha1/openapi_generated.go \
	pkg/apis/workflow/v1alpha1/zz_generated.deepcopy.go \
	manifests/base/crds/full/argoproj.io_workflows.yaml \
	manifests/install.yaml \
	api/openapi-spec/swagger.json \
	api/jsonschema/schema.json \
	docs/fields.md \
	docs/cli/argo.md \
	$(GOPATH)/bin/mockery
	# `go generate ./...` takes around 10s, so we only run on specific packages.
	go generate ./persist/sqldb ./pkg/apiclient/workflow ./server/auth ./server/auth/sso ./workflow/executor
	rm -Rf vendor

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
	go get golang.org/x/tools/cmd/goimports@v0.0.0-20200630154851-b2d8b0336632

pkg/apis/workflow/v1alpha1/generated.proto: $(GOPATH)/bin/go-to-protobuf $(PROTO_BINARIES) $(TYPES)
	[ -e vendor ] || go mod vendor
	${GOPATH}/bin/go-to-protobuf \
		--go-header-file=./hack/custom-boilerplate.go.txt \
		--packages=github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
		--apimachinery-packages=+k8s.io/apimachinery/pkg/util/intstr,+k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/runtime/schema,+k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1,k8s.io/api/policy/v1beta1 \
		--proto-import ./vendor 2>&1 | grep -v 'warning: Import .* is unused'
	touch pkg/apis/workflow/v1alpha1/generated.proto

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

# generate other files for other CRDs
manifests/base/crds/full/argoproj.io_workflows.yaml: $(GOPATH)/bin/controller-gen $(TYPES)
	./hack/crdgen.sh

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
	kustomize build --load_restrictor=none manifests/quick-start/mysql | ./hack/auto-gen-msg.sh > manifests/quick-start-mysql.yaml
	kustomize build --load_restrictor=none manifests/quick-start/postgres | ./hack/auto-gen-msg.sh > manifests/quick-start-postgres.yaml

# lint/test/etc

$(GOPATH)/bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin v1.33.2

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

dist/$(PROFILE).yaml: $(MANIFESTS) $(E2E_MANIFESTS) /usr/local/bin/kustomize
	mkdir -p dist
	kustomize build --load_restrictor=none test/e2e/manifests/$(PROFILE) | sed 's/:latest/:$(VERSION)/' | sed 's/pns/$(E2E_EXECUTOR)/'  > dist/$(PROFILE).yaml

.PHONY: install
install: dist/$(PROFILE).yaml
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
	k3d image import argoproj/argosay:v2
endif
	docker push argoproj/argosay:v2

test/e2e/images/argosay/v2/argosay: test/e2e/images/argosay/v2/main/argosay.go
	cd test/e2e/images/argosay/v2 && GOOS=linux CGO_ENABLED=0 go build -ldflags '-w -s' -o argosay ./main

.PHONY: test-images
test-images:
	$(call docker_pull,argoproj/argosay:v1)
	$(call docker_pull,argoproj/argosay:v2)
	$(call docker_pull,python:alpine3.6)

.PHONY: stop
stop:
	killall argo workflow-controller kubectl || true

$(GOPATH)/bin/goreman:
	go get github.com/mattn/goreman

.PHONY: start
start: stop install controller cli executor-image $(GOPATH)/bin/goreman
	kubectl config set-context --current --namespace=$(KUBE_NAMESPACE)
ifeq ($(RUN_MODE),kubernetes)
	$(MAKE) controller-image cli-image
	kubectl -n $(KUBE_NAMESPACE) scale deploy/workflow-controller --replicas 1
	kubectl -n $(KUBE_NAMESPACE) scale deploy/argo-server --replicas 1
endif
	kubectl -n $(KUBE_NAMESPACE) wait --for=condition=Ready pod --all -l app --timeout 2m
	./hack/port-forward.sh
	# Check dex, minio, postgres and mysql are in hosts file
ifeq ($(AUTH_MODE),sso)
	grep '127.0.0.1[[:blank:]]*dex' /etc/hosts
endif
	grep '127.0.0.1[[:blank:]]*minio' /etc/hosts
	grep '127.0.0.1[[:blank:]]*postgres' /etc/hosts
	grep '127.0.0.1[[:blank:]]*mysql' /etc/hosts
ifeq ($(RUN_MODE),local)
	env DEFAULT_REQUEUE_TIME=$(DEFAULT_REQUEUE_TIME) SECURE=$(SECURE) ALWAYS_OFFLOAD_NODE_STATUS=$(ALWAYS_OFFLOAD_NODE_STATUS) LOG_LEVEL=$(LOG_LEVEL) UPPERIO_DB_DEBUG=$(UPPERIO_DB_DEBUG) VERSION=$(VERSION) AUTH_MODE=$(AUTH_MODE) NAMESPACED=$(NAMESPACED) NAMESPACE=$(KUBE_NAMESPACE) $(GOPATH)/bin/goreman -set-ports=false -logtime=false start
endif

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

.PHONY: test-e2e
test-e2e:
	$(GOTEST) -timeout 10m -count 1 --tags e2e -p 1 --short ./test/e2e

.PHONY: test-cli
test-cli:
	$(GOTEST) -timeout 15m -count 1 --tags cli -p 1 --short ./test/e2e

.PHONY: test-e2e-cron
test-e2e-cron:
	$(GOTEST) -count 1 --tags e2e -parallel 10 -run CronSuite ./test/e2e

.PHONY: smoke
smoke:
	$(GOTEST) -count 1 --tags e2e -p 1 -run SmokeSuite ./test/e2e

# clean

.PHONY: clean
clean:
	go clean
	rm -Rf test-results node_modules vendor dist/* ui/dist

# swagger

pkg/apis/workflow/v1alpha1/openapi_generated.go: $(GOPATH)/bin/openapi-gen $(TYPES)
	openapi-gen \
	  --go-header-file ./hack/custom-boilerplate.go.txt \
	  --input-dirs github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
	  --output-package github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
	  --report-filename pkg/apis/api-rules/violation_exceptions.list

# generates many other files (listers, informers, client etc).
pkg/apis/workflow/v1alpha1/zz_generated.deepcopy.go: $(TYPES)
	bash ${GOPATH}/pkg/mod/k8s.io/code-generator@v0.17.5/generate-groups.sh \
		"deepcopy,client,informer,lister" \
		github.com/argoproj/argo/pkg/client github.com/argoproj/argo/pkg/apis \
		workflow:v1alpha1 \
		--go-header-file ./hack/custom-boilerplate.go.txt

dist/kubernetes.swagger.json:
	@mkdir -p dist
	./hack/recurl.sh dist/kubernetes.swagger.json https://raw.githubusercontent.com/kubernetes/kubernetes/v1.17.5/api/openapi-spec/swagger.json

pkg/apiclient/_.secondary.swagger.json: hack/swagger/secondaryswaggergen.go server/static/files.go pkg/apis/workflow/v1alpha1/openapi_generated.go dist/kubernetes.swagger.json
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
docs/cli/argo.md: $(CLI_PKGS) server/static/files.go hack/cli/main.go
	go run ./hack/cli

.PHONY: validate-examples
validate-examples: api/jsonschema/schema.json
	cd examples && go test

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
