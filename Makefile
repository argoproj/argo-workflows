PACKAGE                := github.com/argoproj/argo

BUILD_DATE             = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT             = $(shell git rev-parse HEAD)
GIT_BRANCH             = $(shell git rev-parse --abbrev-ref=loose HEAD | sed 's/heads\///')
GIT_TAG                = $(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
GIT_TREE_STATE         = $(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)

export DOCKER_BUILDKIT = 1

# docker image publishing options
IMAGE_NAMESPACE       ?= argoproj
ifeq ($(GIT_BRANCH),master)
VERSION               := $(shell cat VERSION)
IMAGE_TAG             := latest
DEV_IMAGE             := true
else
ifeq ($(findstring release,$(GIT_BRANCH)),release)
IMAGE_TAG             := $(VERSION)
DEV_IMAGE             := false
else
VERSION               := $(shell cat VERSION)
IMAGE_TAG             := $(subst /,-,$(GIT_BRANCH))
DEV_IMAGE             := true
endif
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

ARGOEXEC_PKGS    := $(shell echo cmd/argoexec            && go list -f '{{ join .Deps "\n" }}' ./cmd/argoexec/            | grep 'argoproj/argo' | grep -v vendor | cut -c 26-)
CLI_PKGS         := $(shell echo cmd/argo                && go list -f '{{ join .Deps "\n" }}' ./cmd/argo/                | grep 'argoproj/argo' | grep -v vendor | cut -c 26-)
CONTROLLER_PKGS  := $(shell echo cmd/workflow-controller && go list -f '{{ join .Deps "\n" }}' ./cmd/workflow-controller/ | grep 'argoproj/argo' | grep -v vendor | cut -c 26-)
MANIFESTS        := $(shell find manifests          -mindepth 2 -type f)
E2_MANIFESTS     := $(shell find test/e2e/manifests -mindepth 2 -type f)

.PHONY: build
build: clis executor-image controller-image manifests/install.yaml manifests/namespace-install.yaml manifests/quick-start-postgres.yaml manifests/quick-start-mysql.yaml

vendor: Gopkg.toml
	# Get Go dependencies
	rm -Rf .vendor-new
	dep ensure -v

# cli

.PHONY: cli
cli: dist/argo

ui/node_modules: ui/package.json ui/yarn.lock
	# Get UI dependencies
ifeq ($(CI),false)
	yarn --cwd ui install --frozen-lockfile --ignore-optional --non-interactive
else
	mkdir -p ui/node_modules
endif
	touch ui/node_modules

ui/dist/app: ui/node_modules ui/src
	# Build UI
ifeq ($(CI),false)
	yarn --cwd ui build
else
	mkdir -p ui/dist/app
	echo "Built without static files" > ui/dist/app/index.html
endif
	touch ui/dist/app

$(GOPATH)/bin/staticfiles:
	# Install the "staticfiles" tool
	go get bou.ke/staticfiles

cmd/server/static/files.go: ui/dist/app $(GOPATH)/bin/staticfiles
	# Pack UI into a Go file.
	staticfiles -o cmd/server/static/files.go ui/dist/app

dist/argo: vendor cmd/server/static/files.go $(CLI_PKGS)
	go build -v -i -ldflags '${LDFLAGS}' -o dist/argo ./cmd/argo

dist/argo-linux-amd64: vendor cmd/server/static/files.go $(CLI_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-linux-amd64 ./cmd/argo

dist/argo-linux-ppc64le: vendor cmd/server/static/files.go $(CLI_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-linux-ppc64le ./cmd/argo

dist/argo-linux-s390x: vendor cmd/server/static/files.go $(CLI_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-linux-s390x ./cmd/argo

dist/argo-darwin-amd64: vendor cmd/server/static/files.go $(CLI_PKGS)
	CGO_ENABLED=0 GOOS=darwin go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-darwin-amd64 ./cmd/argo

dist/argo-windows-amd64: vendor cmd/server/static/files.go $(CLI_PKGS)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-windows-amd64 ./cmd/argo

.PHONY: cli-image
cli-image: dist/argo-linux-amd64
	# Create CLI image
ifeq ($(DEV_IMAGE),true)
	cp dist/argo-linux-amd64 argo
	docker build -t $(IMAGE_NAMESPACE)/argocli:$(IMAGE_TAG) --target argocli -f Dockerfile.dev .
	rm -f argo
else
	docker build -t $(IMAGE_NAMESPACE)/argocli:$(IMAGE_TAG) --target argocli .
endif

.PHONY: clis
clis: dist/argo-linux-amd64 dist/argo-linux-ppc64le dist/argo-linux-s390x dist/argo-darwin-amd64 dist/argo-windows-amd64 cli-image

# controller

dist/workflow-controller-linux-amd64: vendor $(CONTROLLER_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o dist/workflow-controller-linux-amd64 ./cmd/workflow-controller

.PHONY: controller-image
controller-image: dist/workflow-controller-linux-amd64
	# Create controller image
ifeq ($(DEV_IMAGE),true)
	cp dist/workflow-controller-linux-amd64 workflow-controller
	docker build -t $(IMAGE_NAMESPACE)/workflow-controller:$(IMAGE_TAG) --target workflow-controller -f Dockerfile.dev .
	rm -f workflow-controller
else
	docker build -t $(IMAGE_NAMESPACE)/workflow-controller:$(IMAGE_TAG) --target workflow-controller .
endif

# argoexec

dist/argoexec-linux-amd64: vendor $(ARGOEXEC_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o dist/argoexec-linux-amd64 ./cmd/argoexec

.PHONY: executor-image
executor-image: dist/argoexec-linux-amd64
	# Create executor image
ifeq ($(DEV_IMAGE),true)
	cp dist/argoexec-linux-amd64 argoexec
	docker build -t $(IMAGE_NAMESPACE)/argoexec:$(IMAGE_TAG) --target argoexec -f Dockerfile.dev .
	rm -f argoexec
else
	docker build -t $(IMAGE_NAMESPACE)/argoexec:$(IMAGE_TAG) --target argoexec .
endif

# generation

.PHONY: codegen
codegen:
	# Generate code
	./hack/generate-proto.sh
	./hack/update-codegen.sh
	./hack/update-openapigen.sh
	go run ./hack/gen-openapi-spec/main.go $(VERSION) > ./api/openapi-spec/swagger.json

.PHONY: verify-codegen
verify-codegen:
	# Verify generated code
	./hack/verify-codegen.sh
	./hack/update-openapigen.sh --verify-only
	mkdir -p ./dist
	go run ./hack/gen-openapi-spec/main.go $(VERSION) > ./dist/swagger.json
	diff ./dist/swagger.json ./api/openapi-spec/swagger.json

.PHONY: manifests
manifests: manifests/install.yaml manifests/namespace-install.yaml manifests/quick-start-mysql.yaml manifests/quick-start-postgres.yaml

manifests/install.yaml: $(MANIFESTS)
	env VERSION=$(VERSION) ./hack/update-manifests.sh

manifests/namespace-install.yaml: $(MANIFESTS)
	env VERSION=$(VERSION) ./hack/update-manifests.sh

manifests/quick-start-mysql.yaml: $(MANIFESTS)
	# Create MySQL quick-start manifests
	kustomize build manifests/quick-start/mysql | sed 's/:latest/:$(IMAGE_TAG)/' > manifests/quick-start-mysql.yaml

manifests/quick-start-postgres.yaml: $(MANIFESTS)
	# Create Postgres quick-start manifests
	kustomize build manifests/quick-start/postgres | sed 's/:latest/:$(IMAGE_TAG)/' > manifests/quick-start-postgres.yaml

# lint/test/etc

.PHONY: lint
lint: cmd/server/static/files.go
	# Lint Go files
	golangci-lint run --fix --verbose
ifeq ($(CI),false)
	# Lint UI files
	yarn --cwd ui lint
endif

.PHONY: test
test: cmd/server/static/files.go vendor
	# Run unit tests
ifeq ($(CI),false)
	go test `go list ./... | grep -v 'test/e2e'`
else
	go test -covermode=count -coverprofile=coverage.out `go list ./... | grep -v 'test/e2e'`
endif

test/e2e/manifests/postgres.yaml: $(MANIFESTS) $(E2E_MANIFESTS)
	# Create Postgres e2e manifests
	kustomize build test/e2e/manifests/postgres > test/e2e/manifests/postgres.yaml

dist/postgres.yaml: test/e2e/manifests/postgres.yaml
	# Create Postgres e2e manifests
	cat test/e2e/manifests/postgres.yaml | sed 's/:latest/:$(IMAGE_TAG)/' > dist/postgres.yaml

.PHONY: install-postgres
install-postgres: dist/postgres.yaml
	# Install Postgres quick-start
	kubectl get ns argo || kubectl create ns argo
	kubectl -n argo apply -f dist/postgres.yaml

.PHONY: install
install: install-postgres

.PHONY: start
start: controller-image cli-image install
	# Start development environment
ifeq ($(CI),false)
	make down
	make up
endif
	make executor-image
	# Make the CLI
	make cli
	# Switch to "argo" ns.
	kubectl config set-context --current --namespace=argo

.PHONY: down
down:
	# Scale down
	kubectl -n argo scale deployment/argo-server --replicas 0
	kubectl -n argo scale deployment/workflow-controller --replicas 0
	# Wait for pods to go away, so we don't wait for them to be ready later.
	[ "`kubectl -n argo get pod -l app=argo-server -o name`" = "" ] || kubectl -n argo wait --for=delete pod -l app=argo-server  --timeout 30s
	[ "`kubectl -n argo get pod -l app=workflow-controller -o name`" = "" ] || kubectl -n argo wait --for=delete pod -l app=workflow-controller  --timeout 2m

.PHONY: up
up:
	# Scale up
	kubectl -n argo scale deployment/workflow-controller --replicas 1
	kubectl -n argo scale deployment/argo-server --replicas 1
	# Wait for pods to be ready
	kubectl -n argo wait --for=condition=Ready pod --all -l app --timeout 2m

.PHONY: pf
pf:
	# Start port-forwards
	./hack/port-forward.sh

.PHONY: pf-bg
pf-bg:
	# Start port-forwards in the background
	./hack/port-forward.sh &

.PHONY: logs
logs:
	# Tail logs
	kubectl -n argo logs -f -l app --max-log-requests 10

.PHONY: postgres-cli
postgres-cli:
	kubectl exec -ti `kubectl get pod -l app=postgres -o name|cut -c 5-` -- psql -U postgres

.PHONY: mysql-cli
mysql-cli:
	kubectl exec -ti `kubectl get pod -l app=mysql -o name|cut -c 5-` -- mysql -u mysql -ppassword argo

.PHONY: test-e2e
test-e2e:
	# Run E2E tests
	go test -timeout 20m -v -count 1 -p 1 ./test/e2e/...

.PHONY: smoke
smoke:
	# Run smoke tests
	go test -timeout 45s -v -count 1 -p 1 -run SmokeSuite ./test/e2e

.PHONY: test-api
test-api:
	# Run API tests
	go test -timeout 2m -v -count 1 -p 1 -run ArgoServerSuite ./test/e2e

.PHONY: test-cli
test-cli:
	# Run CLI tests
	go test -timeout 30s -v -count 1 -p 1 -run CliSuite ./test/e2e

# clean

.PHONY: clean
clean:
	# Remove images
	[ "`docker images -q $(IMAGE_NAMESPACE)/argocli:$(IMAGE_TAG)`" = "" ] || docker rmi $(IMAGE_NAMESPACE)/argocli:$(IMAGE_TAG)
	[ "`docker images -q $(IMAGE_NAMESPACE)/argoexec:$(IMAGE_TAG)`" = "" ] || docker rmi $(IMAGE_NAMESPACE)/argoexec:$(IMAGE_TAG)
	[ "`docker images -q $(IMAGE_NAMESPACE)/workflow-controller:$(IMAGE_TAG)`" = "" ] || docker rmi $(IMAGE_NAMESPACE)/workflow-controller:$(IMAGE_TAG)
	# Delete build files
	git clean -fxd -e .idea -e vendor -e ui/node_modules

# pre-push

.git/hooks/pre-push: Makefile
	# Create Git pre-push hook
	echo 'make pre-push' > .git/hooks/pre-push
	chmod +x .git/hooks/pre-push

.PHONY: must-be-clean
must-be-clean:
	# Check everthing has been committed to Git
	@if [ "$(GIT_TREE_STATE)" != "clean" ]; then echo 'git tree state is $(GIT_TREE_STATE)' ; exit 1; fi

.PHONY: pre-commit
pre-commit: test lint codegen manifests start pf-bg smoke test-api test-cli

.PHONY: pre-push
pre-push: must-be-clean pre-commit must-be-clean

# release

.PHONY: prepare-release
prepare-release: pre-release
	# Prepare release
ifeq ($(VERSION),)
	echo "unable to prepare release - VERSION undefined" >&2
	exit 1
endif
ifeq ($(VERSION),latest)
	# No release preparation needed for master branch
else
	# Update VERSION file
	echo $(VERSION) | cut -c 1- > VERSION
	make codegen manifests VERSION=$(VERSION)
	# Commit if any changes
	git diff --quiet || git commit -am "Update manifests to $(VERSION)"
endif

.PHONY: pre-release
pre-release: pre-push
ifeq ($(findstring release,$(GIT_BRANCH)),release)
	# Check we have tagged the latest commit
	@if [ -z "$(GIT_TAG)" ]; then echo 'commit must be tagged to perform release' ; exit 1; fi
	# Check the tag is correct
	@if [ "$(GIT_TAG)" != "v$(VERSION)" ]; then echo 'git tag ($(GIT_TAG)) does not match VERSION (v$(VERSION))'; exit 1; fi
endif

.PHONY: publish
publish:
ifeq ($(VERSION),latest)
ifneq ($(GIT_BRANCH),master)
	echo "you cannot publish latest version unless you are on master" >&2
	exit 1
endif
endif
	# Publish release
	# Push images to Docker Hub
	docker push $(IMAGE_NAMESPACE)/argocli:$(IMAGE_TAG)
	docker push $(IMAGE_NAMESPACE)/argoexec:$(IMAGE_TAG)
	docker push $(IMAGE_NAMESPACE)/workflow-controller:$(IMAGE_TAG)
ifeq ($(SNAPSHOT),false)
	# Push changes to Git
	git push
	git push $(VERSION)
endif

.PHONY: release
release: pre-release build publish
