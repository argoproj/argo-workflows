#!/usr/bin/env bash
set -eu -o pipefail

# order is important, "REPLACEME" -> "workflow"
cat \
    | sed 's/github.com.argoproj.argo_workflows.v3.pkg.apis.workflow.v1alpha1./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/github.com.argoproj.argo_events.pkg.apis.common./io.argoproj.events.v1alpha1./' \
    | sed 's/github.com.argoproj.argo_events.pkg.apis.eventsource.v1alpha1./io.argoproj.events.v1alpha1./' \
    | sed 's/github.com.argoproj.argo_events.pkg.apis.sensor.v1alpha1./io.argoproj.events.v1alpha1./' \
    | sed 's/cronworkflow\./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/event\./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/info\./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/workflowarchive\./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/clusterworkflowtemplate\./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/workflowtemplate\./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/workflow\./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/io.argoproj.REPLACEME.v1alpha1./io.argoproj.workflow.v1alpha1./' \
    | sed 's/k8s.io./io.k8s./' \
    | sed 's/v1alpha1\.v1alpha1\./v1alpha1\./g'
