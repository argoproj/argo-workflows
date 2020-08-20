#!/usr/bin/env bash
set -eu -o pipefail

killall kubectl-autoforward || true

kubectl autoforward &
