#!/bin/bash
set -eu -o pipefail

trap 'rm -Rf vendor' EXIT

if [ "$(ls -t pkg/apis/workflow/v1alpha1/*.go | grep -v 'test\|generated' | head -n1)" -nt pkg/apis/workflow/v1alpha1/generated.proto ]; then
  [ -e vendor ] || go mod vendor
  ${GOPATH}/bin/go-to-protobuf \
    --go-header-file=./hack/custom-boilerplate.go.txt \
    --packages=github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
    --apimachinery-packages=+k8s.io/apimachinery/pkg/util/intstr,+k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/runtime/schema,+k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1,k8s.io/api/policy/v1beta1 \
    --proto-import ./vendor 2>&1 |
    grep -v 'warning: Import .* is unused'
else
  echo "skipping go-to-protobuf: no changes"
fi

find pkg -name '*.proto' ! -name generated.proto | while read -r f; do
  if [ "$(ls -t "$(dirname $f)"/*.pb.go | head -n1)" -nt $f ]; then
    echo "skipping protoc $f: no changes"
    continue
  fi
  echo $f
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
    $f 2>&1 |
    grep -v 'warning: Import .* is unused'
done


