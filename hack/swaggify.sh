#!/usr/bin/env bash
set -eu -o pipefail

cat \
    | sed 's/cronworkflow\.//g' \
    | sed 's/github.com.argoproj.argo.pkg.apis.workflow.v1alpha1.//' \
    | sed 's/google.protobuf.//g' \
    | sed 's/grpc.gateway.runtime.//g' \
    | sed 's/info\.//g' \
    | sed 's/io.k8s.apimachinery.pkg.runtime.//g' \
    | sed 's/k8s.io.api.core.v1.//g' \
    | sed 's/k8s.io.apimachinery.pkg.api.resource.//g' \
    | sed 's/k8s.io.apimachinery.pkg.apis.meta.v1.//g' \
    | sed 's/k8s.io.apimachinery.pkg.util.intstr.//g' \
    | sed 's/workflow\.//g' \
    | sed 's/workflowarchive\.//g' \
    | sed 's/workflowtemplate\.//g'