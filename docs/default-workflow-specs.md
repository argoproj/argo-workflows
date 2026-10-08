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

> v4.2 and after

If teams share a cluster and each namespace needs its own service account or TTL, you can set defaults per namespace, in a ConfigMap in the same namespace as the Workflows.
The controller finds the ConfigMap by its label rather than by a fixed name, so you can call it whatever suits you.
The value under its `workflowDefaults` key has the same shape as the one in the controller config map, so the same document works in either place.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-workflow-defaults
  namespace: my-namespace
  labels:
    workflows.argoproj.io/configmap-type: WorkflowDefaults
data:
  workflowDefaults: |
    spec:
      serviceAccountName: my-namespace-sa
      ttlStrategy:
        secondsAfterCompletion: 60
```

A namespace must have at most one ConfigMap carrying that label.
More than one is an error for every Workflow in that namespace, because there is no sensible way to choose between them.

The controller merges the values rather than replacing them, and the more specific value wins:

`Workflow` > namespace defaults > controller `workflowDefaults`

Namespace defaults may set `spec` fields, plus labels and annotations under `metadata`.
The controller ignores anything else under `metadata`, and `status`, so a namespace ConfigMap cannot put finalizers or owner references on the Workflows of whoever uses that namespace.

`templateReferencing: Strict` doesn't restrict namespace defaults, because it sanitizes only the submitted Workflow spec.
A namespace ConfigMap can therefore set fields a user cannot set directly, such as `serviceAccountName`, `hostNetwork` or `podSpecPatch`, wherever the Workflow and its `WorkflowTemplate` leave them unset.
Write access to that ConfigMap carries the same trust as the controller-level `workflowDefaults`.

A namespace without a labelled ConfigMap has no namespace-level defaults.
The controller treats a ConfigMap that exists but is missing the `workflowDefaults` key, or whose value is not valid YAML, as an error rather than ignoring it, so that defaults never silently fail to apply.
An unrecognized field is an error for the same reason, so the controller reports a misspelling rather than quietly applying nothing.
This also affects Workflows that are already running, not only new ones.
The controller reads the defaults again on every reconcile, so a broken ConfigMap, or a second labelled one, puts every running Workflow in the namespace into `Error`.
Under `templateReferencing: Secure` even a valid edit does this.
On every reconcile the controller re-merges the spec of a running `workflowTemplateRef` Workflow and compares it with the stored one, so any edit to the `spec` defaults that changes the result, including adding one, puts it into `Error`.
