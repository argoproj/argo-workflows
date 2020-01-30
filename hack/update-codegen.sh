#!/bin/bash
set -eux -o pipefail

bash ${GOPATH}/pkg/mod/k8s.io/code-generator@v0.16.7-beta.0/generate-groups.sh \
  "deepcopy,client,informer,lister" \
  github.com/argoproj/argo/pkg/client github.com/argoproj/argo/pkg/apis \
  workflow:v1alpha1 \
  --go-header-file ./hack/custom-boilerplate.go.txt
