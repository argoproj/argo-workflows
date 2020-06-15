#!/bin/bash
set -eux -o pipefail

cd "$(dirname "$0")/.."

del() {
  yq delete $1 $2 > tmp
  mv tmp $1
}

if [ "$(command -v controller-gen)" = "" ]; then
  go install sigs.k8s.io/controller-tools/cmd/controller-gen
fi

if [ "$(command -v yq)" = "" ]; then
  brew install yq
fi

controller-gen crd:trivialVersions=true,maxDescLen=0 paths=./pkg/apis/... output:dir=manifests/base/crds/full

find manifests/base/crds/full -name 'argoproj.io*.yaml' | while read -r file ; do
  # remove junk fields
  del "$file" metadata.annotations
  del "$file" metadata.creationTimestamp
  del "$file" status
  # create minimal
  yq delete "$file" spec.validation > "manifests/base/crds/minimal/$(basename "$file")"
done

