#!/usr/bin/env bash
set -eu -o pipefail

file=test/e2e/kubeconfig

kubectl config view --minify --raw > $file

echo "created/updated $file"