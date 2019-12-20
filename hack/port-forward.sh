#!/usr/bin/env bash
set -eux

killall kubectl || true

trap 'killall kubectl' EXIT

kubectl -n argo port-forward pod/minio 9000:9000 &
kubectl -n argo port-forward svc/argo-server 2746:2746 &
kubectl -n argo port-forward deployment/argo-ui 8001:8001 &

wait