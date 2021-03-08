#!/usr/bin/env bash
set -eux pipefail

cd $(dirname $0)

gen() {
   sed "s/name: E2E/name: $1/" < e2e.yaml.0 |
   sed "s/\${{matrix.test}}/$2/" |
   sed "s/\${{matrix.containerRuntimeExecutor}}/$3/" |
   ../../hack/auto-gen-msg.sh > $2-$3.yaml
}

gen "Test Docker Executor" test-executor docker
gen "Test Emissary Executor" test-executor emissary
gen "Test K8SAPI Executor" test-executor k8sapi
gen "Test Kubelet Executor" test-executor kubelet
gen "Test PNS Executor" test-executor pns

gen "Test CLI" test-cli docker
gen "Test Cron Workflows" test-cron docker
gen "Test Examples" test-examples docker
gen "Test Functionality " test-functional docker
