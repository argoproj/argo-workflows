#!/usr/bin/env bash
set -eu -o pipefail

cat \
    | sed 's/cronworkflow\./io.argoproj.workflow.v1alpha1./g' \
    | sed 's/github.com.argoproj.argo.pkg.apis.workflow.v1alpha1./io.argoproj.workflow.v1alpha1./' \
    | sed 's/info\./io.argoproj.workflow.v1alpha1./g' \
    | sed 's/io.k8s.apimachinery.pkg.runtime./io.k8s.api.core.v1./g' \
    | sed 's/workflow\./io.argoproj.workflow.v1alpha1./g' \
    | sed 's/workflowarchive\./io.argoproj.workflow.v1alpha1./g' \
    | sed 's/workflowtemplate\./io.argoproj.workflow.v1alpha1./g' \
    | sed 's/k8s.io.api.core.v1./io.k8s.api.core.v1./g' \
    | sed 's/k8s.io.apimachinery.pkg.api.resource./io.k8s.api.core.v1./g' \
    | sed 's/k8s.io.apimachinery.pkg.apis.meta.v1./io.k8s.api.core.v1./g' \
    | sed 's/k8s.io.apimachinery.pkg.util.intstr./io.k8s.api.core.v1./g'
