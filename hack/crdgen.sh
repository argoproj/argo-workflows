#!/bin/bash
set -eu -o pipefail

cd "$(dirname "$0")/.."

add_header() {
  cat "$1" | ./hack/auto-gen-msg.sh >tmp
  mv tmp "$1"
}

if [ "$(command -v controller-gen)" = "" ]; then
  go install sigs.k8s.io/controller-tools/cmd/controller-gen
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
