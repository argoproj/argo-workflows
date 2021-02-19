#!/usr/bin/env bash
set -eu -o pipefail

kubectl delete wf -l workflows.argoproj.io/test

grep -lR 'workflows.argoproj.io/test' examples/* | xargs ./dist/argo submit --wait --verify
