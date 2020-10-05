#!/bin/bash
set -eu -o pipefail

cd "$(dirname "$0")/.."

add_header() {
  cat "$1" | ./hack/auto-gen-msg.sh >tmp
  mv tmp "$1"
}

newest_output=$(ls -t manifests/base/crds/full/*.yaml | head -n1)
newest_input=$(ls -t pkg/apis/workflow/v1alpha1/*.go | grep -v 'deepcopy\|generated\|test' | head -n1)

if [ "$newest_input" -nt "$newest_output" ]; then
  echo "Generating CRDs"
  controller-gen crd:trivialVersions=true,maxDescLen=0 paths=./pkg/apis/... output:dir=manifests/base/crds/full

  find manifests/base/crds/full -name 'argoproj.io*.yaml' | while read -r file; do
    echo "Patching ${file}"
    # remove junk fields
    go run ./hack cleancrd "$file"
    add_header "$file"
    # create minimal
    minimal="manifests/base/crds/minimal/$(basename "$file")"
    echo "Creating ${minimal}"
    cp "$file" "$minimal"
    go run ./hack removecrdvalidation "$minimal"
  done
else
  echo "skipping CRDs: no changes"
fi
