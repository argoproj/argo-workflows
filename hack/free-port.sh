#!/bin/sh
set -eu

port=$1

lsof -s TCP:LISTEN -i ":$port" | grep -v PID | awk '{print $2}' | xargs -L 1 kill || true
