#!/bin/bash
set -eux -o pipefail

# @v0.0.0-20200121204235-bf4fb3bd569c
go install k8s.io/kube-openapi/cmd/openapi-gen

openapi-gen \
  --go-header-file ./hack/custom-boilerplate.go.txt \
  --input-dirs github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
  --output-package github.com/argoproj/argo/pkg/apis/workflow/v1alpha1 \
  --report-filename pkg/apis/api-rules/violation_exceptions.list \
  $@
