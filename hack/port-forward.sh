#!/usr/bin/env bash
set -eu -o pipefail

killall kubectl-autoforward || true

if [ "$(command -v kubectl-autoforward)" = "" ]; then
  go install github.com/alexec/kubectl-autoforward
fi

kubectl autoforward &