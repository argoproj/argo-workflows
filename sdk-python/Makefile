SHELL := /bin/bash

BUILDER_IMAGE				= argo-builder
PACKAGE_NAME        = workflows.client
PACKAGE_DESCRIPTION = Python client for Argo Workflows

CURRENT_DIR ?= $(shell pwd)
OUTPUT_DIR  ?= .

define get_branch
$(shell git branch | sed -n '/\* /s///p')
endef

define get_tag
$(shell \
	if [ -z "`git status --porcelain`" ]; then \
		git describe \
			--exact-match \
			--tags HEAD 2>/dev/null || (>&2 echo "Tag has not been created.") \
	fi \
)
endef

define get_tree_state
$(shell \
	if [ -z "`git status --porcelain`" ]; then \
		echo "clean" \
	else \
		echo "dirty" \
	fi
)
endef

GIT_COMMIT     = $(shell git rev-parse HEAD)

GIT_BRANCH     = $(call get_branch)
GIT_TAG        = $(call get_tag)
GIT_TREE_STATE = $(call get_tree_state)

ifeq (${GIT_TAG},)
GIT_TAG = $(shell git rev-parse --abbrev-ref HEAD)
endif

CLIENT_VERSION    ?= $(shell b="${GIT_BRANCH}"; v="$${b/release-/}.0"; echo "$${v:0:5}")

ARGO_VERSION      ?= $(shell cat ARGO_VERSION)
ARGO_API_GROUP    ?= argoproj.io
ARGO_API_VERSION  ?= v1alpha1
ARGO_OPENAPI_SPEC  = openapi/specs/argo-${ARGO_VERSION}.json

KUBERNETES_BRANCH      ?= release-1.16
KUBERNETES_OPENAPI_SPEC = openapi/specs/kubernetes-${KUBERNETES_BRANCH}.json

OPENAPI_SPEC   = openapi/swagger.json
OPENAPI_CONFIG = openapi/custom/config.json

PYPI_REPOSITORY ?= https://upload.pypi.org/legacy/

.PHONY: all
all: clean spec preprocess client
# all: clean validate spec preprocess client

.PHONY: clean
clean:
	-find  ${OUTPUT_DIR}/argo/workflows/client/* -maxdepth 1 -not -name "__*__.py" -exec rm -r {} \;
	-rm -r ${OUTPUT_DIR}/docs/
	-rm -r ${OUTPUT_DIR}/workflows/
	-rm -r ${OUTPUT_DIR}/${PACKAGE_NAME}/

	pushd openapi/ ; git clean -d --force ; popd

.PHONY: patch
patch: SHELL:=/bin/bash
patch:



	sed -i "s/__version__ = \(.*\)/__version__ = \"${CLIENT_VERSION}\"/g" argo/workflows/client/__about__.py

	python setup.py sdist bdist_wheel
	twine check dist/* || (echo "Twine check did not pass. Aborting."; exit 1)

	git commit -a -m ":wrench: Patch ${CLIENT_VERSION}" --signoff
	git tag -a "v${CLIENT_VERSION}" -m "Patch ${CLIENT_VERSION}"

.PHONY: release
release: SHELL:=/bin/bash
release:
	- rm -rf build/ dist/
	- git tag --delete "v${CLIENT_VERSION}"

	$(MAKE) changelog

	sed -i "s/__version__ = \(.*\)/__version__ = \"${CLIENT_VERSION}\"/g" argo/workflows/client/__about__.py

	python setup.py sdist bdist_wheel
	twine check dist/* || (echo "Twine check did not pass. Aborting."; exit 1)

	v=${CLIENT_VERSION}; git commit -a -m ":tada: Release $${v:0:3}" --signoff
	v=${CLIENT_VERSION}; git tag -a "v${CLIENT_VERSION}" -m "Release $${v:0:3}"

validate:
	@echo "Validating version '${CLIENT_VERSION}' on branch '{GIT_BRANCH}'"

	if [ "$(shell python -c \
		"from semantic_version import validate; print( validate('${CLIENT_VERSION}') )" \
	)" != "True" ]; then \
		echo "Invalid version. Aborting."; \
		exit 1; \
	fi

spec:
	# Make sure the folders exist
	mkdir -p openapi/specs/
	mkdir -p openapi/definitions/

	@echo "Collecting API spec for Kubernetes ${ARGO_VERSION}"
	curl -sSL https://raw.githubusercontent.com/kubernetes/kubernetes/${KUBERNETES_BRANCH}/api/openapi-spec/swagger.json \
		-o ${KUBERNETES_OPENAPI_SPEC}

	@echo "Collecting API spec for Argo ${ARGO_VERSION}"
	curl -sSL https://raw.githubusercontent.com/argoproj/argo/v${ARGO_VERSION}/api/openapi-spec/swagger.json \
		-o ${ARGO_OPENAPI_SPEC}

	# @echo "Extracting definitions"
	# jq -r '{ definitions: .definitions }' ${ARGO_OPENAPI_SPEC} \
	# 	> openapi/definitions/argo.json

	# @echo "Merging API definitions"
	# jq -sS '.[0] * .[1]' \
	# 	openapi/definitions/argo.json \
	# 	openapi/definitions/V1Time.json \
	# 	> openapi/definitions.json

	@echo "Creating OpenAPI info"
	echo '{"info": {"title": "Argo Python SDK", "description": "${PACKAGE_DESCRIPTION}", "version": "${ARGO_VERSION}"}}' | jq -r '.' \
		> openapi/info.json

	# @echo "Process OpenAPI paths"
	# jinja2 openapi/custom/paths.json --format=json --strict \
	# 	-Dargo_api_group=${ARGO_API_GROUP} \
	# 	-Dargo_api_version=${ARGO_API_VERSION} \
	# 	> openapi/paths.json

	@echo "Creating OpenAPI spec"
	jq -s '.[0]' \
	 	openapi/info.json \
	 	> ${OPENAPI_SPEC}
	# @echo "Creating OpenAPI spec"
	# jq -s '.[0] + .[1] + .[2] + .[3] + .[4]' \
	# 	openapi/custom/version.json \
	#  	openapi/info.json \
	#  	openapi/custom/security.json \
	#  	openapi/paths.json \
	#  	openapi/definitions.json \
	#  	> ${OPENAPI_SPEC}


preprocess:
	@echo "Preprocessing API specs"
	python3 scripts/preprocess.py -i ${ARGO_OPENAPI_SPEC} \
		-d 'io.argoproj.workflow' \
		-d 'cronio.argoproj.workflow' \
		-d 'io.k8s.api.core' \
		-d 'io.k8s.apimachinery.pkg.apis.meta' \
		-o ${OPENAPI_SPEC} >/dev/null

	# Replace empty references
	sed -i -e '/"$$ref"/ s/io.argoproj.workflow.//' ${OPENAPI_SPEC}


	# Patch DAGTask template requirement
  # No longer needed
	# This is an unpleasant workaround, since OpenAPI 2.0 does not allow `oneOf`
	# jq -r '.definitions."v1alpha1.DAGTask".required = ["name"]' ${OPENAPI_SPEC} |\
	# sponge ${OPENAPI_SPEC}

.PHONY:client
client: clean
	-find  ${OUTPUT_DIR}/argo/workflows/client/* -maxdepth 1 -not -name "__*__.py" -exec rm -r {} \;
	-rm -r ${OUTPUT_DIR}/docs/
	-rm -r ${OUTPUT_DIR}/workflows/
	-rm -r ${OUTPUT_DIR}/${PACKAGE_NAME}/

	@echo "Generating Argo ${ARGO_VERSION} client"

	CLIENT_VERSION=${CLIENT_VERSION} \
	KUBERNETES_BRANCH=${KUBERNETES_BRANCH} \
	PACKAGE_NAME=${PACKAGE_NAME} \
		./scripts/generate_client.sh ${OUTPUT_DIR} ${OPENAPI_SPEC} ${OPENAPI_CONFIG}

changelog:
	RELEASE_VERSION=${CLIENT_VERSION} ./scripts/generate_changelog.sh

.PHONY:builder_image
builder_image:
	docker build -f builder_image/Dockerfile -t ${BUILDER_IMAGE} .

builder_make:
	docker run -w `pwd` -it --entrypoint make --rm -v `pwd`:`pwd` -v /var/run/docker.sock:/var/run/docker.sock ${BUILDER_IMAGE}

builder_release:
	docker run -w `pwd` -it --entrypoint make --rm -v `pwd`:`pwd` -v /var/run/docker.sock:/var/run/docker.sock ${BUILDER_IMAGE} release
