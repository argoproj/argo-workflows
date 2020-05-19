#!/bin/bash
set -eux -o pipefail

source $(dirname $0)/library.sh

header "generating proto files"

if [ ! -d "${REPO_ROOT}/vendor" ]; then
  export GO111MODULE="on"
  go mod vendor
fi

make_fake_paths

export GOPATH="${FAKE_GOPATH}"
export GO111MODULE="off"


cd "${FAKE_REPOPATH}"
go run ${FAKE_REPOPATH}/vendor/k8s.io/code-generator/cmd/go-to-protobuf \
    --go-header-file=./hack/custom-boilerplate.go.txt \
    --packages=github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
    --apimachinery-packages=+k8s.io/apimachinery/pkg/util/intstr,+k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/runtime/schema,+k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1,k8s.io/api/policy/v1beta1  \
    --proto-import ./vendor

# Following 2 proto files are needed
mkdir -p ${GOPATH}/src/google/api    
curl -Ls https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/v1.14.5/third_party/googleapis/google/api/annotations.proto -o ${GOPATH}/src/google/api/annotations.proto
curl -Ls https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/v1.14.5/third_party/googleapis/google/api/http.proto -o ${GOPATH}/src/google/api/http.proto

for f in $(find pkg -name '*.proto'); do
    protoc \
        -I /usr/local/include \
        -I . \
        -I ./vendor \
        -I ${GOPATH}/src \
        --include_imports \
        --gogofast_out=plugins=grpc:${GOPATH}/src \
        --grpc-gateway_out=logtostderr=true:${GOPATH}/src \
        --swagger_out=logtostderr=true,fqn_for_swagger_name=true:. \
        $f
done

