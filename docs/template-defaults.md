# Template Defaults

> v3.1 and after

## Introduction

`TemplateDefaults` feature enables the user to configure the default template values in workflow spec level that will apply to all the templates in the workflow. If the template has a value that also has a default value in `templateDefault`, the Template's value will take precedence. These values will be applied during the runtime. Template values and default values are merged using Kubernetes strategic merge patch. To check whether and how list values are merged, inspect the `patchStrategy` and `patchMergeKey` tags in the [workflow definition](https://github.com/argoproj/argo-workflows/blob/main/pkg/apis/workflow/v1alpha1/workflow_types.go).

## Configuring `templateDefaults` in `WorkflowSpec`

For example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: template-defaults-example
spec:
  entrypoint: main
  templateDefaults:
    timeout: 30s   # timeout value will be applied to all templates
    retryStrategy: # retryStrategy value will be applied to all templates
      limit: 2
  templates:
  - name: main
    container:
      image: busybox
```

[template defaults example](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/template-defaults.yaml)

## Configuring `templateDefaults` in Controller Level

Operator can configure the `templateDefaults` in [workflow defaults](default-workflow-specs.md). This `templateDefault` will be applied to all the workflow which runs on the controller.

The following would be specified in the Config Map:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  # Default values that will apply to all Workflows from this controller, unless overridden on the Workflow-level
  workflowDefaults: |
    metadata:
      annotations:
        argo: workflows
      labels:
        foo: bar
    spec:
      ttlStrategy:
        secondsAfterSuccess: 5
      templateDefaults:
        timeout: 30s
```
