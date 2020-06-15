#!/bin/bash
set -eux -o pipefail

if [ "$(command -v go-to-protobuf)" = "" ]; then
  go get k8s.io/code-generator/cmd/go-to-protobuf@v0.17.3
fi

bash ${GOPATH}/pkg/mod/k8s.io/code-generator@v0.17.3/generate-groups.sh \
  "deepcopy,client,informer,lister" \
  github.com/argoproj/argo/pkg/client github.com/argoproj/argo/pkg/apis \
  workflow:v1alpha1 \
  --go-header-file ./hack/custom-boilerplate.go.txt
