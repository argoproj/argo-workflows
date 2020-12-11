#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname $0)/.."

kustomize build manifests/quick-start/base/prometheus | kubectl -n argo apply --force -f -
kubectl -n argo apply -f test/e2e/stress/many-massive-workflows.yaml

argo submit \
  -n argo \
  --from workflowtemplate/many-massive-workflows \
  -p workflows="500" \
  -p nodes="1" \
  -p sleep="1m" \
  --watch
