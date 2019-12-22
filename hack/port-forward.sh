#!/usr/bin/env bash
set -eux -o pipefail

killall kubectl || true

trap 'killall kubectl' EXIT

echo "minio on 9000"
kubectl -n argo port-forward pod/minio 9000:9000 &

echo "argo-server on 2746"
argo_server=$(kubectl -n argo get pod -l app=argo-server -o name)
if [ "$argo_server" != "" ]; then
  kubectl -n argo port-forward svc/argo-server 2746:2746 &
fi

argo_ui=$(kubectl -n argo get pod -l app=argo-ui -o name)
if [ "$argo_ui" != "" ]; then
  echo "argo-ui on 8001"
  kubectl -n argo port-forward deployment/argo-ui 8001:8001 &
fi

postgres=$(kubectl -n argo get pod -l app=postgres -o name)
if [ "$postgres" != "" ]; then
  echo "postgres on 5432"
  kubectl -n argo port-forward "$postgres" 5432:5432 &
fi

mysql=$(kubectl -n argo get pod -l app=mysql -o name)
if [ "$mysql" != "" ]; then
  echo "mysql on 3306"
  kubectl -n argo port-forward "$mysql" 3306:3306 &
fi

wait