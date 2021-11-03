#!/usr/bin/env bash
set -eu -o pipefail

# Load the configmaps that contains the parameter values used for certain examples.
kubectl apply -f examples/configmaps/simple-parameters-configmap.yaml

grep -lR 'workflows.argoproj.io/test' examples/* | while read f ; do
  kubectl delete wf -l workflows.argoproj.io/test
  kubectl create -f $f
  kubectl wait --for=condition=Completed wf -l workflows.argoproj.io/test --timeout 10s
  test Succeeded = "$(kubectl get wf -l workflows.argoproj.io/test -o 'jsonpath={.status.phase}')"
done
