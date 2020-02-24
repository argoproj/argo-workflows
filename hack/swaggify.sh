#!/usr/bin/env bash
set -eu -o pipefail

# order is important, "REPLACEME" -> "workflow"
cat \
    | sed 's/github.com.argoproj.argo.pkg.apis.workflow.v1alpha1./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/workflow\./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/cronworkflow\./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/info\./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/workflowarchive\./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/workflowtemplate\./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/io.argoproj.REPLACEME.v1alpha1./io.argoproj.workflow.v1alpha1./' \
    | sed 's/io.k8s.apimachinery.pkg.runtime./io.k8s.api.core.v1./' \
    | sed 's/k8s.io.api.core.v1./io.k8s.api.core.v1./' \
    | sed 's/k8s.io.apimachinery.pkg.api.resource./io.k8s.api.core.v1./' \
    | sed 's/k8s.io.apimachinery.pkg.apis.meta.v1./io.k8s.api.core.v1./' \
    | sed 's/k8s.io.apimachinery.pkg.util.intstr./io.k8s.api.core.v1./'
