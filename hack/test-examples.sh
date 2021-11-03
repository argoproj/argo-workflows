#!/usr/bin/env bash
set -eu -o pipefail

# Load the configmaps that contains the parameter values used for certain examples.
kubectl apply -f examples/configmaps/simple-parameters-configmap.yaml

grep -lR 'workflows.argoproj.io/test' examples/* | while read f ; do
  kubectl delete wf -l workflows.argoproj.io/test
  echo "RUN $f"
  kubectl create -f $f
  name=$(kubectl get wf -o name)
  kubectl wait --for=condition=Completed $name --timeout 10s
  phase="$(kubectl get $name -o 'jsonpath={.status.phase}')"
  test Succeeded == $phase
done
