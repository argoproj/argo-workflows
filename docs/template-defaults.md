#Template Defaults
> v3.1 and after

## Introduction

`TemplateDefaults` feature enables user to configure the default template values in workflow spec level that will apply to all templates in the workflow.
These values will be applied in runtime. If template has a value that also has a default value in `templateDefault`, the Template's value will take precedence. 


## Configuring `templateDefaults` in WorkflowSpec

For example:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  labels:
    workflows.argoproj.io/archive-strategy: "false"
spec:
  entrypoint: whalesay
  templateDefaults:
    timeout: 30s   # timeout value will be applied to all templates
    retryStrategy: # retryStrategy value will be applied to all templates
      limit: 2
    container:
      imagePullPolicy: "Never" # imagePullPolicy will only be applied to container type templates
  templates:
  - name: whalesay
    activeDeadlineSeconds: 150
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
```

## Configuring `templateDefaults` in Controller level
Operator can configure the `templateDefaults` in [workflowDefaults](default-workflow-specs.md). This `templateDefault` will be applied all workflow which runs on these controller.

The following would be specified in the Config Map:

```yaml
# This file describes the config settings available in the workflow controller configmap
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