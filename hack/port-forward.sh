#!/usr/bin/env bash
set -eu -o pipefail

pf() {
  set -eu -o pipefail
  resource=$1
  port=$2
  dest_port=${3:-"$port"}
  ./hack/free-port.sh $port
  echo "port-forward $resource $port"
  kubectl -n argo port-forward "svc/$resource" "$port:$dest_port" &
	until lsof -i ":$port" > /dev/null ; do sleep 1 ; done
}

wait-for() {
  set -eu -o pipefail
  echo "wait-for $1"
  kubectl -n argo wait --timeout 2m --for=condition=Available deploy/$1
}


dex=$(kubectl -n argo get pod -l app=dex -o name)
if [[ "$dex" != "" ]]; then
  wait-for dex
  pf dex 5556
fi

postgres=$(kubectl -n argo get pod -l app=postgres -o name)
if [[ "$postgres" != "" ]]; then
  wait-for postgres
  pf postgres 5432
fi

mysql=$(kubectl -n argo get pod -l app=mysql -o name)
if [[ "$mysql" != "" ]]; then
	wait-for mysql
  pf mysql 3306
fi

if [[ "$(kubectl -n argo get pod -l app=argo-server -o name)" != "" ]]; then
  wait-for argo-server
  pf argo-server 2746
fi

if [[ "$(kubectl -n argo get pod -l app=workflow-controller -o name)" != "" ]]; then
  wait-for workflow-controller
  pf workflow-controller-metrics 9090
  if [[ "$(kubectl -n argo get svc workflow-controller-pprof -o name)" != "" ]]; then
    pf workflow-controller-pprof 6060
  fi
fi

if [[ "$(kubectl -n argo get pod -l app=prometheus -o name)" != "" ]]; then
  wait-for prometheus
  pf prometheus 9091 9090
fi

azurite=$(kubectl -n argo get pod -l app=azurite -o name)
if [[ "$azurite" != "" ]]; then
  wait-for azurite
  pf azurite 10000
fi

# forward MinIO last, so we can just wait for port 9000, and know that all ports are ready
wait-for minio
pf minio 9000
pf minio 9001