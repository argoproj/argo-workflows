#!/bin/bash
set -eu -o pipefail

newest_output=$(ls -t pkg/client/listers/workflow/v1alpha1/*.go | head -n1)
newest_input=$(ls -t pkg/apis/workflow/v1alpha1/*.go | grep -v 'generated\|test' | head -n1)

if [ "$newest_input" -nt "$newest_output" ]; then
  echo "running generate-groups.sh"
  bash ${GOPATH}/pkg/mod/k8s.io/code-generator@v0.17.5/generate-groups.sh \
    "deepcopy,client,informer,lister" \
    github.com/argoproj/argo/pkg/client github.com/argoproj/argo/pkg/apis \
    workflow:v1alpha1 \
    --go-header-file ./hack/custom-boilerplate.go.txt
else
  echo "skipping generate-groups.sh: no changes"
fi
