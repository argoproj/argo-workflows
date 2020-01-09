PACKAGE                := github.com/argoproj/argo

BUILD_DATE             = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT             = $(shell git rev-parse HEAD)
GIT_BRANCH             = $(shell git rev-parse --abbrev-ref HEAD)
GIT_TAG                = $(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
GIT_TREE_STATE         = $(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)

# docker image publishing options
IMAGE_NAMESPACE       = argoproj

export DOCKER_BUILDKIT = 1

# version must be  branch name or  vX.Y.Z
ifeq ($(GIT_BRANCH), master)
VERSION               ?= latest
else
VERSION               ?= $(GIT_BRANCH)
endif

# perform static compilation
STATIC_BUILD          ?= true
CI                    ?= false

override LDFLAGS += \
  -X ${PACKAGE}.version=$(VERSION) \
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
ifeq ($(VERSION),$(GIT_BRANCH))
	SNAPSHOT=true
endif

ARGOEXEC_PKGS    := $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/argoexec/|grep 'argoproj/argo'|grep -v vendor|cut -c 26-)
ARGO_SERVER_PKGS := $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/server/|grep 'argoproj/argo'|grep -v vendor|cut -c 26-)
CLI_PKGS         := $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/argo/|grep 'argoproj/argo'|grep -v vendor|cut -c 26-)
CONTROLLER_PKGS  := $(shell go list -f '{{ join .Deps "\n" }}' ./cmd/workflow-controller/|grep 'argoproj/argo'|grep -v vendor|cut -c 26-)

.PHONY: build
build: clis controller-image executor-image argo-server

vendor: Gopkg.toml
	dep ensure -v -vendor-only

# cli

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
cli-image: dist/argo-linux-amd64
	cp dist/argo-linux-amd64 argo
	docker build -t $(IMAGE_NAMESPACE)/argocli:$(VERSION) --target argocli .
	rm -f argo

.PHONY: clis
clis: dist/argo-linux-amd64 dist/argo-linux-ppc64le dist/argo-linux-s390x dist/argo-darwin-amd64 dist/argo-windows-amd64 cli-image

# controller

dist/workflow-controller-linux-amd64: vendor $(CONTROLLER_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o dist/workflow-controller-linux-amd64 ./cmd/workflow-controller

.PHONY: controller-image
controller-image: dist/workflow-controller-linux-amd64
	cp dist/workflow-controller-linux-amd64 workflow-controller
	docker build -t $(IMAGE_NAMESPACE)/workflow-controller:$(VERSION) --target workflow-controller .
	rm -f workflow-controller

# argo-server

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

$(GOPATH)/bin/staticfiles:
	go get bou.ke/staticfiles

cmd/server/static/files.go: ui/dist/app $(GOPATH)/bin/staticfiles
	staticfiles -o cmd/server/static/files.go ui/dist/app

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
	docker build -t $(IMAGE_NAMESPACE)/argo-server:$(VERSION) -f Dockerfile --target argo-server .
	rm -f argo-server

.PHONY: argo-server
argo-server: dist/argo-server-linux-amd64 dist/argo-server-linux-ppc64le dist/argo-server-linux-s390x dist/argo-server-darwin-amd64 dist/argo-server-windows-amd64

# argoexec

dist/argoexec-linux-amd64:
	go build -v -i -ldflags '${LDFLAGS}' -o dist/argoexec-linux-amd64 ./cmd/argoexec

.PHONY: executor-image
executor-image: dist/argoexec-linux-amd64
	cp dist/argoexec-linux-amd64 argoexec
	docker build -t $(IMAGE_NAMESPACE)/argoexec:$(VERSION) --target argoexec .
	rm -f argoexec

# generation

.PHONY: codegen
codegen:
	./hack/generate-proto.sh
	./hack/update-codegen.sh
	./hack/update-openapigen.sh
	go run ./hack/gen-openapi-spec/main.go $(VERSION) > ./api/openapi-spec/swagger.json

.PHONY: verify-codegen
verify-codegen:
	./hack/verify-codegen.sh
	./hack/update-openapigen.sh --verify-only
	mkdir -p ./dist
	go run ./hack/gen-openapi-spec/main.go $(VERSION) > ./dist/swagger.json
	diff ./dist/swagger.json ./api/openapi-spec/swagger.json

.PHONY: manifests
manifests:
	env VERSION=$(VERSION) ./hack/update-manifests.sh

# lint/test/etc

.PHONY: lint
lint: cmd/server/static/files.go
	golangci-lint run --fix --verbose --config golangci.yml
ifeq ($(CI),false)
	yarn --cwd ui lint
endif

.PHONY: test
test: cmd/server/static/files.go vendor
ifeq ($(CI),false)
	go test `go list ./... | grep -v 'test/e2e'`
else
	go test -covermode=count -coverprofile=coverage.out `go list ./... | grep -v 'test/e2e'`
endif

.PHONY: start
start: controller-image argo-server-image executor-image
	env INSTALL_CLI=0 VERSION=dev ./install.sh
	# Scale down in preparation for re-configuration.
	make down
	# Change to use a "dev" tag and enable debug logging.
	kubectl -n argo patch deployment/workflow-controller --type json --patch '[{"op": "replace", "path": "/spec/template/spec/containers/0/imagePullPolicy", "value": "Never"}, {"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "argoproj/workflow-controller:$(VERSION)"}, {"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": ["--loglevel", "debug", "--executor-image", "argoproj/argoexec:$(VERSION)", "--executor-image-pull-policy", "Never"]}]'
	# TODO Turn on the workflow compression, hopefully to shake out some bugs.
	# kubectl -n argo patch deployment/workflow-controller --type json --patch '[{"op": "add", "path": "/spec/template/spec/containers/0/env", "value": [{"name": "MAX_WORKFLOW_SIZE", "value": "1000"}]}]'
	kubectl -n argo patch deployment/argo-server --type json --patch '[{"op": "replace", "path": "/spec/template/spec/containers/0/imagePullPolicy", "value": "Never"}, {"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": "argoproj/argo-server:$(VERSION)"}, {"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": ["--loglevel", "debug", "--auth-type", "client"]}]'
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

# clean

.PHONY: clean
clean:
	git clean -fxd -e .idea -e vendor -e ui/node_modules

# release

.PHONY: prepare-release
prepare-release: pre-release
ifeq ($(VERSION),)
	echo "unable to prepare release - VERSION undefined"
	exit 1
endif
ifeq ($(GIT_BRANCH),master)
	echo "no release preparation needed for master branch"
else
	echo "preparing release $(VERSION)"
	echo $(VERSION) | cut -c 1- > VERSION
	make codegen manifests VERSION=$(VERSION)
	# only commit if changes
	git diff --quiet || git commit -am "Update manifests to $(VERSION)"
endif

.PHONY: must-be-clean
must-be-clean:
	@if [ "$(GIT_TREE_STATE)" != "clean" ]; then echo 'git tree state is $(GIT_TREE_STATE)' ; exit 1; fi

.PHONY: pre-release
pre-release: must-be-clean test lint codegen manifests must-be-clean
ifeq ($(SNAPSHOT),false)
	@if [ -z "$(GIT_TAG)" ]; then echo 'commit must be tagged to perform release' ; exit 1; fi
	@if [ "$(GIT_TAG)" != "v$(VERSION)" ]; then echo 'git tag ($(GIT_TAG)) does not match VERSION (v$(VERSION))'; exit 1; fi
endif

.PHONY: publish
publish:
	docker push $(IMAGE_NAMESPACE)/argocli:$(VERSION)
	docker push $(IMAGE_NAMESPACE)/argoexec:$(VERSION)
	docker push $(IMAGE_NAMESPACE)/argo-server:$(VERSION)
	docker push $(IMAGE_NAMESPACE)/workflow-controller:$(VERSION)
	git push $(GIT_BRANCH) --tags

.PHONY: release
release: pre-release build publish
