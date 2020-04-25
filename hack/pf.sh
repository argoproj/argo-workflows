#!/usr/bin/env bash
set -eu -o pipefail

pf() {
  set -eu -o pipefail
  name=$1
  resource=$2
  port=$3
  lsof -i ":$port" | grep -v PID | awk '{print $2}' | xargs kill || true
  info "$name on http://localhost:$port"
  kubectl -n argo port-forward "$resource" "$port:$port"
}

info() {
    echo '[INFO] ' "$@"
}

pf MinoIO pod/minio 9000 &

postgres=$(kubectl -n argo get pod -l app=postgres -o name)
if [[ "$postgres" != "" ]]; then
  pf Postgres "$postgres" 5432 &
fi

mysql=$(kubectl -n argo get pod -l app=mysql -o name)
if [[ "$mysql" != "" ]]; then
  pf MySQL "$mysql" 3306 &
fi

wait