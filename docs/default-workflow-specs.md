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

## Namespace-Level Default Workflow Values

Defaults can also be set per namespace, in a ConfigMap named `workflow-defaults` in the same namespace as the Workflows they apply to. The value under the `workflowDefaults` key has the same shape as the one in the controller config map, so the same document works in either place.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-defaults
  namespace: my-namespace
data:
  workflowDefaults: |
    spec:
      serviceAccountName: my-namespace-sa
      ttlStrategy:
        secondsAfterCompletion: 60
```

A namespace can have at most one, because the name is fixed and Kubernetes already guarantees that names are unique within a namespace.

Values are merged rather than replaced, and the more specific value wins:

`Workflow` > namespace `workflow-defaults` > controller `workflowDefaults`

A namespace without the ConfigMap has no namespace-level defaults. A ConfigMap that exists but is missing the `workflowDefaults` key, or whose value is not valid YAML, is an error rather than being ignored, so that defaults never silently fail to apply.
