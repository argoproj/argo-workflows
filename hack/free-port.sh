#!/bin/sh
set -eux

port=$1

lsof -i ":$port" | grep -v PID | awk '{print $2}' | xargs kill || true


