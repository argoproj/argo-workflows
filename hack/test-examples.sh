#!/usr/bin/env bash
set -eu -o pipefail

# Load the configmaps that contains the parameter values used for certain examples.
kubectl apply -f examples/configmaps/simple-parameters-configmap.yaml

echo "Checking for banned images..."
grep -lR 'workflows.argoproj.io/test' examples/*  | while read f ; do
  echo " - $f"
  test 0 == $(grep -o 'image: .*' $f | grep -cv 'argoproj/argosay:v2\|python:alpine3.6\|busybox')
done

trap 'kubectl get wf' EXIT

grep -lR 'workflows.argoproj.io/test' examples/* | while read f ; do
  echo "Running $f..."
  name=$(kubectl create -f $f -o name)

  echo "Waiting for completion of $f..."
  kubectl wait --for=condition=Completed $name
  phase="$(kubectl get $name -o 'jsonpath={.status.phase}')"
  echo " -> $phase"
  test Succeeded == $phase

  echo "Deleting $f..."
  kubectl delete $name
done
