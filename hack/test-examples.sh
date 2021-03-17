#!/usr/bin/env bash
set -eu -o pipefail

./dist/argo delete -l workflows.argoproj.io/test

grep -lR 'workflows.argoproj.io/test' examples/* | while read f ; do
  ./dist/argo submit --watch --verify $f
done
