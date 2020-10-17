#!/bin/bash
set -eu -o pipefail

f=$1

trap 'rm -Rf vendor' EXIT
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
