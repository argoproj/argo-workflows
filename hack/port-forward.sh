#!/usr/bin/env bash
set -eu -o pipefail

killall kubectl || true

info() {
    echo '[INFO] ' "$@"
}

info "MinIO on http://localhost:9000"
kubectl -n argo port-forward pod/minio 9000:9000 &

info "Prometheus on http://localhost:9090"
kubectl -n argo port-forward deploy/workflow-controller 9090:9090 &

argo_server=$(kubectl -n argo get pod -l app=argo-server -o name)
if [[ "$argo_server" != "" ]]; then
  info "Argo Server on http://localhost:2746"
  kubectl -n argo port-forward svc/argo-server 2746:2746 &
fi

postgres=$(kubectl -n argo get pod -l app=postgres -o name)
if [[ "$postgres" != "" ]]; then
  info "Postgres on http://localhost:5432"
  kubectl -n argo port-forward "$postgres" 5432:5432 &
fi

mysql=$(kubectl -n argo get pod -l app=mysql -o name)
if [[ "$mysql" != "" ]]; then
  info "MySQL on http://localhost:3306"
  kubectl -n argo port-forward "$mysql" 3306:3306 &
fi

wait