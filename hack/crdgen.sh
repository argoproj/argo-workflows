#!/bin/bash
set -eu -o pipefail

cd "$(dirname "$0")/.."

add_header() {
  cat "$1" | ./hack/auto-gen-msg.sh >tmp
  mv tmp "$1"
}

oldest_output=$(ls -t manifests/base/crds/full/*.yaml | tail -n1)
newest_input=$(ls -t pkg/apis/workflow/v1alpha1/*.go | grep -v 'deepcopy\|generated\|test'| head -n1)

if [ "$oldest_output" -nt "$newest_input" ]; then
  echo "skipping CRDs: no changes"
  exit
fi

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
