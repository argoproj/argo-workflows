
BUILD_DATE             = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT             = $(shell git rev-parse HEAD)
GIT_REMOTE             = origin
GIT_BRANCH             = $(shell git rev-parse --abbrev-ref=loose HEAD | sed 's/heads\///')
GIT_TAG                = $(shell git describe --exact-match --tags HEAD 2>/dev/null)
GIT_TREE_STATE         = $(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)

export DOCKER_BUILDKIT = 1

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
GIT_LATEST_TAG := $(shell git describe --abbrev=0 --tags)
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

# perform static compilation
STATIC_BUILD          ?= true
CI                    ?= false
DB                    ?= postgres
K3D                   := $(shell if [ "`kubectl config current-context`" = "k3s-default" ]; then echo true; else echo false; fi)
ARGO_TOKEN            = $(shell kubectl -n argo get secret -o name | grep argo-server | xargs kubectl -n argo get -o jsonpath='{.data.token}' | base64 --decode)

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
# the sort puts _.primary first in the list
SWAGGER_FILES    := $(shell find pkg -name '*.swagger.json' | sort)

.PHONY: build
build: status clis executor-image controller-image manifests/install.yaml manifests/namespace-install.yaml manifests/quick-start-postgres.yaml manifests/quick-start-mysql.yaml

.PHONY: status
status:
	# GIT_TAG=$(GIT_TAG), GIT_BRANCH=$(GIT_BRANCH), GIT_TREE_STATE=$(GIT_TREE_STATE), VERSION=$(VERSION), DEV_IMAGE=$(DEV_IMAGE)

.PHONY: vendor
vendor: go.mod go.sum
	go mod download

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

$(HOME)/go/bin/staticfiles:
	# Install the "staticfiles" tool
	go get bou.ke/staticfiles

server/static/files.go: $(HOME)/go/bin/staticfiles ui/dist/app
	# Pack UI into a Go file.
	staticfiles -o server/static/files.go ui/dist/app

dist/argo: vendor server/static/files.go $(CLI_PKGS)
	go build -v -i -ldflags '${LDFLAGS}' -o dist/argo ./cmd/argo

dist/argo-linux-amd64: vendor server/static/files.go $(CLI_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-linux-amd64 ./cmd/argo

dist/argo-linux-ppc64le: vendor server/static/files.go $(CLI_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-linux-ppc64le ./cmd/argo

dist/argo-linux-s390x: vendor server/static/files.go $(CLI_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-linux-s390x ./cmd/argo

dist/argo-darwin-amd64: vendor server/static/files.go $(CLI_PKGS)
	CGO_ENABLED=0 GOOS=darwin go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-darwin-amd64 ./cmd/argo

dist/argo-windows-amd64: vendor server/static/files.go $(CLI_PKGS)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -v -i -ldflags '${LDFLAGS}' -o dist/argo-windows-amd64 ./cmd/argo

.PHONY: cli-image
cli-image: dist/cli-image

dist/cli-image: dist/argo-linux-amd64
	# Create CLI image
ifeq ($(DEV_IMAGE),true)
	cp dist/argo-linux-amd64 argo
	docker build -t $(IMAGE_NAMESPACE)/argocli:$(VERSION) --target argocli -f Dockerfile.dev .
	rm -f argo
else
	docker build -t $(IMAGE_NAMESPACE)/argocli:$(VERSION) --target argocli .
endif
	touch dist/cli-image
ifeq ($(K3D),true)
	k3d import-images $(IMAGE_NAMESPACE)/argocli:$(VERSION)
endif

.PHONY: clis
clis: dist/argo-linux-amd64 dist/argo-linux-ppc64le dist/argo-linux-s390x dist/argo-darwin-amd64 dist/argo-windows-amd64 cli-image

# controller

dist/workflow-controller-linux-amd64: vendor $(CONTROLLER_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o dist/workflow-controller-linux-amd64 ./cmd/workflow-controller

.PHONY: controller-image
controller-image: dist/controller-image

dist/controller-image: dist/workflow-controller-linux-amd64
	# Create controller image
ifeq ($(DEV_IMAGE),true)
	cp dist/workflow-controller-linux-amd64 workflow-controller
	docker build -t $(IMAGE_NAMESPACE)/workflow-controller:$(VERSION) --target workflow-controller -f Dockerfile.dev .
	rm -f workflow-controller
else
	docker build -t $(IMAGE_NAMESPACE)/workflow-controller:$(VERSION) --target workflow-controller .
endif
	touch dist/controller-image
ifeq ($(K3D),true)
	k3d import-images $(IMAGE_NAMESPACE)/workflow-controller:$(VERSION)
endif

# argoexec

dist/argoexec-linux-amd64: vendor $(ARGOEXEC_PKGS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -i -ldflags '${LDFLAGS}' -o dist/argoexec-linux-amd64 ./cmd/argoexec

.PHONY: executor-image
executor-image: dist/executor-image

dist/executor-image: dist/argoexec-linux-amd64
	# Create executor image
ifeq ($(DEV_IMAGE),true)
	cp dist/argoexec-linux-amd64 argoexec
	docker build -t $(IMAGE_NAMESPACE)/argoexec:$(VERSION) --target argoexec -f Dockerfile.dev .
	rm -f argoexec
else
	docker build -t $(IMAGE_NAMESPACE)/argoexec:$(VERSION) --target argoexec .
endif
	touch dist/executor-image
ifeq ($(K3D),true)
	k3d import-images $(IMAGE_NAMESPACE)/argoexec:$(VERSION)
endif

# generation

.PHONY: codegen
codegen:
	# Generate code
	# We need the vendor folder for compatibility
	go mod vendor

	./hack/generate-proto.sh
	./hack/update-codegen.sh
	make api/openapi-spec/swagger.json
	find . -path '*/mocks/*' -type f -not -path '*/vendor/*' -exec ./hack/update-mocks.sh {} ';'

	rm -rf ./vendor
	go mod tidy


.PHONY: manifests
manifests: status manifests/install.yaml manifests/namespace-install.yaml manifests/quick-start-mysql.yaml manifests/quick-start-postgres.yaml manifests/quick-start-no-db.yaml test/e2e/manifests/postgres.yaml test/e2e/manifests/mysql.yaml test/e2e/manifests/no-db.yaml

# we use a different file to ./VERSION to force updating manifests after a `make clean`
dist/MANIFESTS_VERSION:
	mkdir -p dist
	echo $(MANIFESTS_VERSION) > dist/MANIFESTS_VERSION

manifests/install.yaml: dist/MANIFESTS_VERSION $(MANIFESTS)
	kustomize build manifests/cluster-install | sed "s/:latest/:$(MANIFESTS_VERSION)/" | ./hack/auto-gen-msg.sh > manifests/install.yaml

manifests/namespace-install.yaml: dist/MANIFESTS_VERSION $(MANIFESTS)
	kustomize build manifests/namespace-install | sed "s/:latest/:$(MANIFESTS_VERSION)/" | ./hack/auto-gen-msg.sh > manifests/namespace-install.yaml

manifests/quick-start-no-db.yaml: dist/MANIFESTS_VERSION $(MANIFESTS)
	kustomize build manifests/quick-start/no-db | sed "s/:latest/:$(MANIFESTS_VERSION)/" | ./hack/auto-gen-msg.sh > manifests/quick-start-no-db.yaml

manifests/quick-start-mysql.yaml: dist/MANIFESTS_VERSION $(MANIFESTS)
	kustomize build manifests/quick-start/mysql | sed "s/:latest/:$(MANIFESTS_VERSION)/" | ./hack/auto-gen-msg.sh > manifests/quick-start-mysql.yaml

manifests/quick-start-postgres.yaml: dist/MANIFESTS_VERSION $(MANIFESTS)
	kustomize build manifests/quick-start/postgres | sed "s/:latest/:$(MANIFESTS_VERSION)/" | ./hack/auto-gen-msg.sh > manifests/quick-start-postgres.yaml

# lint/test/etc

.PHONY: lint
lint: server/static/files.go
	# Tidy Go modules
	go mod tidy
	# Lint Go files
	golangci-lint run --fix --verbose
ifeq ($(CI),false)
	# Lint UI files
	yarn --cwd ui lint
endif

.PHONY: test
test: server/static/files.go vendor
	# Run unit tests
ifeq ($(CI),false)
	go test `go list ./... | grep -v 'test/e2e'`
else
	go test -covermode=count -coverprofile=coverage.out `go list ./... | grep -v 'test/e2e'`
endif

test/e2e/manifests/postgres.yaml: $(MANIFESTS) $(E2E_MANIFESTS)
	# Create Postgres e2e manifests
	kustomize build test/e2e/manifests/postgres | ./hack/auto-gen-msg.sh > test/e2e/manifests/postgres.yaml

dist/postgres.yaml: test/e2e/manifests/postgres.yaml
	# Create Postgres e2e manifests
	cat test/e2e/manifests/postgres.yaml | sed 's/:latest/:$(VERSION)/' | sed 's/pns/$(E2E_EXECUTOR)/' > dist/postgres.yaml

test/e2e/manifests/no-db/overlays/argo-server-deployment.yaml: test/e2e/manifests/postgres/overlays/argo-server-deployment.yaml
test/e2e/manifests/no-db/overlays/argo-server-deployment.yaml:
	cat test/e2e/manifests/postgres/overlays/argo-server-deployment.yaml | ./hack/auto-gen-msg.sh > test/e2e/manifests/no-db/overlays/argo-server-deployment.yaml

test/e2e/manifests/no-db/overlays/workflow-controller-deployment.yaml: test/e2e/manifests/postgres/overlays/workflow-controller-deployment.yaml
test/e2e/manifests/no-db/overlays/workflow-controller-deployment.yaml:
	cat test/e2e/manifests/postgres/overlays/workflow-controller-deployment.yaml | ./hack/auto-gen-msg.sh > test/e2e/manifests/no-db/overlays/workflow-controller-deployment.yaml

test/e2e/manifests/no-db.yaml: $(MANIFESTS) $(E2E_MANIFESTS) test/e2e/manifests/no-db/overlays/argo-server-deployment.yaml test/e2e/manifests/no-db/overlays/workflow-controller-deployment.yaml
	# Create no DB e2e manifests
	kustomize build test/e2e/manifests/no-db | ./hack/auto-gen-msg.sh > test/e2e/manifests/no-db.yaml

dist/no-db.yaml: test/e2e/manifests/no-db.yaml
	# Create no DB e2e manifests
	# We additionlly disable ALWAY_OFFLOAD_NODE_STATUS
	cat test/e2e/manifests/no-db.yaml | sed 's/:latest/:$(VERSION)/' | sed 's/pns/$(E2E_EXECUTOR)/' | sed 's/"true"/"false"/' > dist/no-db.yaml

test/e2e/manifests/mysql/overlays/argo-server-deployment.yaml: test/e2e/manifests/postgres/overlays/argo-server-deployment.yaml
test/e2e/manifests/mysql/overlays/argo-server-deployment.yaml:
	cat test/e2e/manifests/postgres/overlays/argo-server-deployment.yaml | ./hack/auto-gen-msg.sh > test/e2e/manifests/mysql/overlays/argo-server-deployment.yaml

test/e2e/manifests/mysql/overlays/workflow-controller-deployment.yaml: test/e2e/manifests/postgres/overlays/workflow-controller-deployment.yaml
test/e2e/manifests/mysql/overlays/workflow-controller-deployment.yaml:
	cat test/e2e/manifests/postgres/overlays/workflow-controller-deployment.yaml | ./hack/auto-gen-msg.sh > test/e2e/manifests/mysql/overlays/workflow-controller-deployment.yaml

test/e2e/manifests/mysql.yaml: $(MANIFESTS) $(E2E_MANIFESTS) test/e2e/manifests/mysql/overlays/argo-server-deployment.yaml test/e2e/manifests/mysql/overlays/workflow-controller-deployment.yaml
	# Create MySQL e2e manifests
	kustomize build test/e2e/manifests/mysql | ./hack/auto-gen-msg.sh > test/e2e/manifests/mysql.yaml

dist/mysql.yaml: test/e2e/manifests/mysql.yaml
	# Create MySQL e2e manifests
	cat test/e2e/manifests/mysql.yaml | sed 's/:latest/:$(VERSION)/' | sed 's/pns/$(E2E_EXECUTOR)/' > dist/mysql.yaml

.PHONY: install
install: dist/postgres.yaml dist/mysql.yaml dist/no-db.yaml
	# Install Postgres quick-start
	kubectl get ns argo || kubectl create ns argo
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
test-images: dist/cowsay-v1 dist/bitnami-kubectl-1.15.3-ol-7-r165 dist/python-alpine3.6

dist/cowsay-v1:
	docker build -t cowsay:v1 test/e2e/images/cowsay
ifeq ($(K3D),true)
	k3d import-images cowsay:v1
endif
	touch dist/cowsay-v1

dist/bitnami-kubectl-1.15.3-ol-7-r165:
	docker pull bitnami/kubectl:1.15.3-ol-7-r165
	touch dist/bitnami-kubectl-1.15.3-ol-7-r165

dist/python-alpine3.6:
	docker pull python:alpine3.6
	touch dist/python-alpine3.6

.PHONY: start
start: status controller-image cli-image executor-image install
	# Start development environment
ifeq ($(CI),false)
	make down
endif
	make up
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
	# Token
	make env

# this is a convenience to get the login token, you can use it as follows
#   eval $(make env)
#   argo token

.PHONY: env
env:
	export ARGO_SERVER=localhost:2746
	export ARGO_TOKEN=$(ARGO_TOKEN)

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
	go test -timeout 20m -v -count 1 -p 1 ./test/e2e/...

.PHONY: smoke
smoke: test-images
	# Run smoke tests
	go test -timeout 2m -v -count 1 -p 1 -run SmokeSuite ./test/e2e

.PHONY: test-api
test-api: test-images
	# Run API tests
	go test -timeout 3m -v -count 1 -p 1 -run ArgoServerSuite ./test/e2e

.PHONY: test-cli
test-cli: test-images cli
	# Run CLI tests
	go test -timeout 1m -v -count 1 -p 1 -run CLISuite ./test/e2e
	go test -timeout 1m -v -count 1 -p 1 -run CLIWithServerSuite ./test/e2e

# clean

.PHONY: clean
clean:
	# Delete pre-go 1.3 vendor
	rm -Rf vendor
	# Delete build files
	rm -Rf dist ui/dist

# swagger

$(HOME)/go/bin/swagger:
	go get github.com/go-swagger/go-swagger/cmd/swagger

api/openapi-spec/swagger.json: $(HOME)/go/bin/swagger $(SWAGGER_FILES) dist/MANIFESTS_VERSION hack/swaggify.sh
	swagger mixin -c 412 $(SWAGGER_FILES) | sed 's/VERSION/$(MANIFESTS_VERSION)/' | ./hack/swaggify.sh > api/openapi-spec/swagger.json
	# Override ParallelSteps definition to match overridden serialization/deserialization logic.  https://github.com/argoproj/argo/issues/2454
	cat api/openapi-spec/swagger.json | jq '.definitions["io.argoproj.workflow.v1alpha1.ParallelSteps"] = { "type": "object", "additionalProperties": { "type": "array", "items": { "$ref": "#/definitions/io.argoproj.workflow.v1alpha1.WorkflowStep" } } }' > api/openapi-spec/swagger.json.tweaked
	mv api/openapi-spec/swagger.json.tweaked api/openapi-spec/swagger.json

# pre-push

.PHONY: pre-commit
pre-commit: test lint codegen manifests start pf-bg smoke test-api test-cli

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
