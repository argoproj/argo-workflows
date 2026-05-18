# Workflow Restrictions

> v2.9 and after

## Introduction

As the administrator of the controller, you may want to limit which types of Workflows your users can run.
Workflow Restrictions allow you to set requirements for all Workflows.

## Available Restrictions

* `templateReferencing: Strict`: Only process Workflows using `workflowTemplateRef`. You can use this to require usage of WorkflowTemplates, disallowing arbitrary Workflow execution.
* `templateReferencing: Secure`: Same as `Strict` _plus_ enforce that a referenced WorkflowTemplate hasn't changed between operations. If a running Workflow's underlying WorkflowTemplate changes, the Workflow will error out.

## Allowed Workflow Fields Under `templateReferencing`

When `templateReferencing` is set to `Strict` or `Secure`, the submitted `Workflow` may only set fields that are explicitly allowed on top of the referenced `WorkflowTemplate`. Any other field present on the submission is rejected and the Workflow errors out.

This prevents users from overriding security-sensitive fields defined in the `WorkflowTemplate` (such as `serviceAccountName`, `securityContext`, `volumes`, `hostNetwork`, `podSpecPatch`, or injecting additional `templates`) via their submission.

The allow-listed fields are:

* `arguments`
* `entrypoint`
* `shutdown`
* `suspend`
* `activeDeadlineSeconds`
* `priority`
* `ttlStrategy`
* `podGC`
* `volumeClaimGC`
* `archiveLogs`
* `workflowMetadata`
* `workflowTemplateRef`
* `metrics`
* `artifactGC`

All other fields on the submitted `Workflow` spec must be defined on the referenced `WorkflowTemplate` instead.

## Setting Workflow Restrictions

You can add `workflowRestrictions` in the [`workflow-controller-configmap`](./workflow-controller-configmap.yaml).

For example, to specify that Workflows may only run with `workflowTemplateRef`:

```yaml
# This file describes the config settings available in the workflow controller configmap
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  workflowRestrictions: |
    templateReferencing: Strict
```
