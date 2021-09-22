#!/usr/bin/env bash
set -eu -o pipefail

./dist/argo delete -l workflows.argoproj.io/test

# Load the configmaps that contains the parameter values used for certain examples.
kubectl apply -f examples/configmaps/simple-parameters-configmap.yaml

grep -lR 'workflows.argoproj.io/test' examples/* | while read f ; do
  ./dist/argo submit --watch --verify $f
done
