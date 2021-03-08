#!/usr/bin/env bash
set -eux pipefail

cd $(dirname $0)

gen() {
   sed "s/name: Test/name: $1/" < e2e.yaml.0 |
   sed "s/\${{matrix.test}}/$2/" |
   sed "s/\${{matrix.containerRuntimeExecutor}}/$3/" |
   ../../hack/auto-gen-msg.sh > $2-$3.yaml
}

gen "Test Docker Executor" test-executor docker
gen "Test Emissary Executor" test-executor emissary
gen "Test K8SAPI Executor" test-executor k8sapi
gen "Test Kubelet Executor" test-executor kubelet
gen "Test PNS Executor" test-executor pns

gen "CLI Tests" test-cli docker
gen "Cron Tests" test-cron docker
gen "Functional Tests" test-functional docker
