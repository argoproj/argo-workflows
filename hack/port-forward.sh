#!/usr/bin/env bash
set -eux -o pipefail

echo "ALEX 3"
pf() {
  set -eu -o pipefail
  echo "ALEX 4"
  name=$1
  resource=$2
  port=$3
  dest_port=${4:-"$port"}
  echo "ALEX 5"
  ./hack/free-port.sh $port
  echo "ALEX 6"
  info "$name on http://localhost:$port"
  kubectl -n argo port-forward "$resource" "$port:$dest_port" > /dev/null &
  echo "ALEX 7"
  # wait until port forward is established
	until lsof -i ":$port" > /dev/null ; do sleep 1s ; done
}

echo "ALEX 8"
info() {
    echo '[INFO] ' "$@"
}
echo "ALEX 9"

kubectl -n argo wait --timeout 1m --for=condition=Available deploy minio
pf MinIO svc/minio 9000
echo "ALEX 10"

dex=$(kubectl -n argo get pod -l app=dex -o name)
if [[ "$dex" != "" ]]; then
  kubectl -n argo wait --timeout 1m --for=condition=Available deploy dex
  pf DEX svc/dex 5556
fi
echo "ALEX 11"

postgres=$(kubectl -n argo get pod -l app=postgres -o name)
if [[ "$postgres" != "" ]]; then
  kubectl -n argo wait --timeout 1m --for=condition=Available deploy postgres
  pf Postgres "$postgres" 5432
fi
echo "ALEX 12"

mysql=$(kubectl -n argo get pod -l app=mysql -o name)
if [[ "$mysql" != "" ]]; then
	kubectl -n argo wait --timeout 1m --for=condition=Available deploy mysql
  pf MySQL "$mysql" 3306
fi
echo "ALEX 13"

if [[ "$(kubectl -n argo get pod -l app=argo-server -o name)" != "" ]]; then
  kubectl -n argo wait --for=condition=Available deploy argo-server
  pf "Argo Server" svc/argo-server 2746
fi
echo "ALEX 14"

if [[ "$(kubectl -n argo get pod -l app=workflow-controller -o name)" != "" ]]; then
  kubectl -n argo wait --for=condition=Available deploy workflow-controller
  pf "Workflow Controller Metrics" svc/workflow-controller-metrics 9090
  if [[ "$(kubectl -n argo get svc -l app=workflow-controller-pprof -o name)" != "" ]]; then
    pf "Workflow Controller PProf" svc/workflow-controller-pprof 6060
  fi
fi
echo "ALEX 15"

if [[ "$(kubectl -n argo get pod -l app=prometheus -o name)" != "" ]]; then
  kubectl -n argo wait --for=condition=Available deploy prometheus
  pf "Prometheus Server" svc/prometheus 9091 9090
fi

echo "ALEX 16"