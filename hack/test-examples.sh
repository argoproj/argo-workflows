#!/usr/bin/env bash
set -eu -o pipefail

# Load the configmaps that contains the parameter values used for certain examples.
kubectl apply -f examples/configmaps/simple-parameters-configmap.yaml

echo "Checking for banned images..."
grep -lR 'workflows.argoproj.io/test' examples/*  | while read f ; do
  echo " - $f"
  test 0 == $(grep -o 'image: .*' $f | grep -cv 'argoproj/argosay:v2\|python:alpine3.6')
done

trap 'kubectl get wf' EXIT

grep -lR 'workflows.argoproj.io/test' examples/* | while read f ; do
  kubectl delete workflow -l workflows.argoproj.io/test
  echo "Running $f..."
  kubectl create -f $f
  name=$(kubectl get workflow -o name)
  kubectl wait --for=condition=Completed $name
  phase="$(kubectl get $name -o 'jsonpath={.status.phase}')"
  echo " -> $phase"
  test Succeeded == $phase
done
