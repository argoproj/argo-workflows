#!/usr/bin/env bash
set -eu -o pipefail

pf() {
  set -eu -o pipefail
  name=$1
  resource=$2
  port=$3
  dest_port=${4:-"$port"}
  pid=$(lsof -i ":$port" | grep -v PID | awk '{print $2}' || true)
  if [ "$pid" != "" ]; then
    kill $pid
  fi
  kubectl -n argo port-forward "$resource" "$port:$dest_port" > /dev/null &
  # wait until port forward is established
	until lsof -i ":$port" > /dev/null ; do sleep 1s ; done
  info "$name on http://localhost:$port"
}

info() {
    echo '[INFO] ' "$@"
}

killall kubectl || true

if [[ "$(kubectl -n argo get pod -l app=minio -o name)" != "" ]]; then
  pf MinIO deploy/minio 9000
fi

dex=$(kubectl -n argo get pod -l app=dex -o name)
if [[ "$dex" != "" ]]; then
  pf DEX svc/dex 5556
fi

postgres=$(kubectl -n argo get pod -l app=postgres -o name)
if [[ "$postgres" != "" ]]; then
  pf Postgres "$postgres" 5432
fi

mysql=$(kubectl -n argo get pod -l app=mysql -o name)
if [[ "$mysql" != "" ]]; then
  pf MySQL "$mysql" 3306
fi

if [[ "$(kubectl -n argo get pod -l app=argo-server -o name)" != "" ]]; then
  pf "Argo Server" svc/argo-server 2746
fi

if [[ "$(kubectl -n argo get pod -l app=workflow-controller -o name)" != "" ]]; then
  pf "Workflow Controller Metrics" svc/workflow-controller-metrics 9090
  pf "Workflow Controller PProf" svc/workflow-controller-pprof 6060
fi

if [[ "$(kubectl -n argo get pod -l app=prometheus -o name)" != "" ]]; then
  pf "Prometheus Server" svc/prometheus 9091 9090
fi

