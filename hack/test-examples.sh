#!/usr/bin/env bash
set -eu -o pipefail

# Grant admin privileges for the default service account so we could test the examples that submit k8s resources.
kubectl create rolebinding default-admin --clusterrole=admin --serviceaccount=argo:default -n argo

grep -lR 'workflows.argoproj.io/test' examples/* | while read f ; do
  ./dist/argo delete -l workflows.argoproj.io/test
  ./dist/argo submit --watch --verify $f
done
