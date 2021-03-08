#!/usr/bin/env bash
set -eux pipefail

cd $(dirname $0)

gen() {
   sed "s/name: E2E/name: $1/" < e2e.yaml.0 | sed "s/name: Test/name: $1/" | sed "s/\${{matrix.test}}/$2/" | sed "s/\${{matrix.containerRuntimeExecutor}}/$3/" > $2-$3.yaml
}

gen "Test Docker Executor" test-executor docker