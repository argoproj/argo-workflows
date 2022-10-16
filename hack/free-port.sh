#!/usr/bin/env bash
set -eu -o pipefail

port=$1

pids=$(lsof -t -s TCP:LISTEN -i ":$port" || true)

if [ "$pids" != "" ]; then
  kill $pids
fi
