#!/bin/bash
set -eux -o pipefail

MANIFESTS_VERSION=$1

source $(dirname $0)/library.sh

header "running swagger generator"

ensure_vendor
make_fake_paths

export GOPATH="${FAKE_GOPATH}"
export GO111MODULE="off"

cd "${FAKE_REPOPATH}"

SWAGGER_CMD="go run ${FAKE_REPOPATH}/vendor/github.com/go-swagger/go-swagger/cmd/swagger"
SWAGGER_FILES=$(find pkg/apiclient -name '*.swagger.json' | env LC_COLLATE=C sort)

if [ ! -e dist/kubernetes.swagger.json ]; then
  ./hack/recurl.sh dist/kubernetes.swagger.json https://raw.githubusercontent.com/kubernetes/kubernetes/release-1.15/api/openapi-spec/swagger.json
fi

go run ${FAKE_REPOPATH}/vendor/k8s.io/kube-openapi/cmd/openapi-gen \
          --go-header-file ./hack/custom-boilerplate.go.txt \
          --input-dirs github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
          --output-package github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
          --report-filename pkg/apis/api-rules/violation_exceptions.list

go run ./hack secondaryswaggergen

${SWAGGER_CMD} mixin -c 680 ${SWAGGER_FILES} | sed "s/VERSION/$MANIFESTS_VERSION/g" | ./hack/swaggify.sh > dist/swagger.json

go run ./hack kubeifyswagger dist/swagger.json dist/kubeified.swagger.json

${SWAGGER_CMD} flatten --with-flatten minimal --with-flatten remove-unused dist/kubeified.swagger.json > api/openapi-spec/swagger.json
${SWAGGER_CMD} validate api/openapi-spec/swagger.json

go test ./api/openapi-spec

