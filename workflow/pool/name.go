package pool

import (
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// NameMaxLength is the longest pool name: `<name>-<10-char hash>` plus
// the 17 characters Kubernetes adds to a Deployment's pod names stays within
// a 63-character hostname.
const NameMaxLength = 35

// Name is the pool's name: pool.name, or by default
// `<workflowtemplate>-<template>` for a template from a WorkflowTemplate or
// ClusterWorkflowTemplate and `<template>` otherwise, lowercased, with `.`
// replaced by `-`, cut to NameMaxLength and trailing `-` trimmed.
func Name(pool *wfv1.Pool, workflowTemplateName, templateName string) string {
	if pool != nil && pool.Name != "" {
		return pool.Name
	}
	name := templateName
	if workflowTemplateName != "" {
		name = workflowTemplateName + "-" + templateName
	}
	name = strings.ReplaceAll(strings.ToLower(name), ".", "-")
	if len(name) > NameMaxLength {
		name = name[:NameMaxLength]
	}
	return strings.TrimRight(name, "-")
}
