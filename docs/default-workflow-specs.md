# Default Workflow Spec

> v2.7 and after

## Introduction

Default Workflow spec values can be set on [the controller config map](./workflow-controller-configmap.md) that will apply to all Workflows executed from said controller.
Default values are most useful for config-related fields that you want to repeat across all Workflows, such as garbage collection.
If a Workflow has a value that also has a default value set in the config map, the Workflow's value will take precedence.

## Setting Default Workflow Values

Default Workflow values can be specified by adding them under the `workflowDefaults` key in the `workflow-controller-configmap`.
Any values under `Workflow.metadata` and `Workflow.spec` can be set as workflow defaults.
See the [Field Reference](./fields.md#workflow) for full details of `ObjectMeta` and `WorkflowSpec`.

For example, to specify default values that would partially produce the following `Workflow`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: gc-ttl-
  annotations:
    argo: workflows
  labels:
    foo: bar
spec:
  ttlStrategy:
    secondsAfterSuccess: 5     # Time to live after workflow is successful
  parallelism: 3
```

The following would be specified in the Config Map:

```yaml
# This file describes the config settings available in
# the workflow controller configmap
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  # Default values that will apply to all Workflows from
  # this controller, unless overridden on the Workflow-level
  workflowDefaults: |
    metadata:
      annotations:
        argo: workflows
      labels:
        foo: bar
    spec:
      ttlStrategy:
        secondsAfterSuccess: 5
      parallelism: 3

```

## Setting Default CronWorkflow Values

> v4.1 and after

Default `CronWorkflow` spec values can be specified under the `cronWorkflowDefaults` key in the `workflow-controller-configmap`.
These apply to all `CronWorkflows` executed from the controller, such as `concurrencyPolicy`, `startingDeadlineSeconds`, `successfulJobsHistoryLimit`, `failedJobsHistoryLimit` and `timezone`.
If a `CronWorkflow` sets a value that also has a default, the `CronWorkflow`'s value takes precedence.
See the [Field Reference](./fields.md#cronworkflowspec) for the full list of `CronWorkflowSpec` fields.

Defaults are applied to the controller's in-memory copy during reconciliation and are never written back to the `CronWorkflow`, so they do not conflict with GitOps tools that manage the resource.

For example, to keep at most four successful and four failed child Workflows for every `CronWorkflow`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  cronWorkflowDefaults: |
    concurrencyPolicy: "Replace"
    successfulJobsHistoryLimit: 4
    failedJobsHistoryLimit: 4
    timezone: "America/New_York"
```
