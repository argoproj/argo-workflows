#!/bin/bash
set -eu -o pipefail

cd "$(dirname "$0")/../.." # up to repo root

add_header() {
  cat "$1" | ./hack/manifests/auto-gen-msg.sh >tmp
  mv tmp "$1"
}

controller-gen crd:generateEmbeddedObjectMeta=true paths=./pkg/apis/... output:dir=manifests/base/crds/full

find manifests/base/crds/full -name 'argoproj.io*.yaml' | while read -r file; do
  # remove junk fields
  go run ./hack/manifests cleancrd "$file"
  add_header "$file"
  # create minimal
  minimal="manifests/base/crds/minimal/$(basename "$file")"
  echo "Creating minimal CRD file: ${minimal}"
  cp "$file" "$minimal"
  go run ./hack/manifests minimizecrd "$minimal"
done
