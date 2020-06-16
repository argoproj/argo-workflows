#!/bin/bash
set -eu -o pipefail

cd "$(dirname "$0")/.."

del() {
  yq delete $1 $2 >tmp
  mv tmp "$1"
}

add_header() {
  cat "$1" | ./hack/auto-gen-msg.sh >tmp
  mv tmp "$1"
}

if [ "$(command -v controller-gen)" = "" ]; then
  go install sigs.k8s.io/controller-tools/cmd/controller-gen
fi

export PATH=$PATH:dist

if [ "$(command -v yq)" = "" ]; then
  if [ "$(uname)" = "Darwin" ]; then
    brew install yq
  else
    ./hack/recurl.sh dist/yq https://github.com/mikefarah/yq/releases/download/3.3.2/yq_linux_amd64
  fi
fi

echo "Generating CRDs"
controller-gen crd:trivialVersions=true,maxDescLen=0 paths=./pkg/apis/... output:dir=manifests/base/crds/full

find manifests/base/crds/full -name 'argoproj.io*.yaml' | while read -r file; do
  echo "Patching ${file}"
  # remove junk fields
  del "$file" metadata.annotations
  del "$file" metadata.creationTimestamp
  del "$file" status
  add_header "$file"
  # create minimal
  minimal="manifests/base/crds/minimal/$(basename "$file")"
  echo "Creating ${minimal}"
  yq delete "$file" spec.validation >"$minimal"
done
