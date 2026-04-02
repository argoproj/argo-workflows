#!/usr/bin/env bash
set -eu -o pipefail

go install github.com/kitproj/kubeauto@v0.0.7

kubeauto -p 0
