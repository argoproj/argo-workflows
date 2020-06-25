#!/bin/bash
set -eux -o pipefail
go get k8s.io/code-generator/cmd/go-to-protobuf@v0.16.7-beta.0
go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v1.12.2
go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@v1.12.2
go get github.com/gogo/protobuf/protoc-gen-gogo@v1.3.1
go get github.com/gogo/protobuf/protoc-gen-gogofast@v1.3.1
go get github.com/gogo/protobuf/gogoproto@v1.3.1
go get golang.org/x/tools/cmd/goimports@v0.0.0-20200428211428-0c9eba77bc32
go install k8s.io/code-generator/cmd/go-to-protobuf
go-to-protobuf \
    --go-header-file=./hack/custom-boilerplate.go.txt \
    --packages=github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
    --apimachinery-packages=+k8s.io/apimachinery/pkg/util/intstr,+k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/runtime/schema,+k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1,k8s.io/api/policy/v1beta1  \
    --proto-import ./vendor

for f in $(find pkg -name '*.proto'); do
    protoc \
        -I /usr/local/include \
        -I . \
        -I ./vendor \
        -I ${GOPATH}/src \
        -I ${GOPATH}/pkg/mod/github.com/gogo/protobuf@v1.3.1/gogoproto \
        -I ${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.12.2/third_party/googleapis \
        --include_imports \
        --gogofast_out=plugins=grpc:${GOPATH}/src \
        --grpc-gateway_out=logtostderr=true:${GOPATH}/src \
        --swagger_out=logtostderr=true,fqn_for_swagger_name=true:. \
        $f
done
