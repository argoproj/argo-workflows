#!/usr/bin/env bash
set -eu -o pipefail

file=test/e2e/kubeconfig

kubectl config view --minify --raw | sed "s/127.0.0.1/$(hostname)/g" > $file

echo "created/updated $file"