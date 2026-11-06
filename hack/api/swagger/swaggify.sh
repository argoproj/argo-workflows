#!/usr/bin/env bash
set -eu -o pipefail

# order is important, "REPLACEME" -> "workflow"
cat \
    | sed 's/github.com.argoproj.argo_workflows.v4.pkg.apis.workflow.v1alpha1./io.argoproj.REPLACEME.v1alpha1./' \
    | sed 's/github.com.argoproj.argo_events.pkg.apis.events.v1alpha1./io.argoproj.events.v1alpha1./' \
    | sed 's/[A-Z][a-zA-Z]*Service\.\([A-Z]\)/\1/g' `# protoc-gen-openapiv2 fqn naming prefixes nested request-body messages with "FooService."; strip it. Unanchored: also rewrites any prose containing such a token.` \
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
