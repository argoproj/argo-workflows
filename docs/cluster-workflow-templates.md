# Cluster Workflow Templates

> v2.8 and after

## Introduction

`ClusterWorkflowTemplates` are cluster scoped `WorkflowTemplates`. `ClusterWorkflowTemplate` 
can be created cluster scoped like `ClusterRole` and can be accessed all namespaces in the cluster. 

`WorkflowTemplate` documentation [link](./workflow-template.md)

## Defining `ClusterWorkflowTemplate`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-workflow-template-whalesay-template
spec:
  templates:
  - name: whalesay-template
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
---
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-workflow-template-random-fail-template
spec:
  templates:
  - name: random-fail-template
    retryStrategy:
      limit: 10
    container:
      image: python:alpine3.6
      command: [python, -c]
      # fail with a 66% probability
      args: ["import random; import sys; exit_code = random.choice([0, 1, 1]); sys.exit(exit_code)"]
---
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-workflow-template-inner-steps
spec:
  templates:
  - name: whalesay-template
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
  - name: inner-steps
    steps:
    - - name: inner-hello1
        templateRef:
          name: cluster-workflow-template-whalesay-template
          template: whalesay-template
          clusterscope: true
        arguments:
          parameters:
          - name: message
            value: "inner-hello1"
    - - name: inner-hello2a
        templateRef:
          name: cluster-workflow-template-whalesay-template
          template: whalesay-template
          clusterscope: true
        arguments:
          parameters:
          - name: message
            value: "inner-hello2a"
      - name: inner-hello2b
        templateRef:
          name: cluster-workflow-template-whalesay-template
          template: whalesay-template
          clusterscope: true
        arguments:
          parameters:
          - name: message
            value: "inner-hello2b"

```

## Referencing other `ClusterWorkflowTemplates`

You can reference `templates` from another `ClusterWorkflowTemplates` using a `templateRef` field with `clusterScope: true` .
Just as how you reference other `templates` within the same `Workflow`, you should do so from a `steps` or `dag` template.

Here is an example from a `steps` template:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    steps:                              # You should only reference external "templates" in a "steps" or "dag" "template".
      - - name: call-whalesay-template
          templateRef:                  # You can reference a "template" from another "WorkflowTemplate or ClusterWorkflowTemplate" using this field
            name: cluster-workflow-template-whalesay-template   # This is the name of the "WorkflowTemplate or ClusterWorkflowTempalte" CRD that contains the "template" you want
            template: whalesay-template # This is the name of the "template" you want to reference
            clusterScope: true          # This field indicates this templateRef is pointing ClusterWorkflowTemplate
          arguments:                    # You can pass in arguments as normal
            parameters:
            - name: message
              value: "hello world"
```

You can also do so similarly with a `dag` template:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    dag:
      tasks:
        - name: call-whalesay-template
          templateRef:
            name: workflow-template-1
            template: whalesay-template
            clusterScope: true
          arguments:
            parameters:
            - name: message
              value: "hello world"
```
## Managing `ClusterWorkflowTemplates`

### CLI

You can create some example templates as follows:

```
argo cluster-template create https://raw.githubusercontent.com/argoproj/argo/master/examples/cluster-workflow-template/clustertemplates.yaml
```

The submit a workflow using one of those templates:

```
argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/cluster-workflow-template/cluster-wftmpl-dag.yaml
```

### `kubectl`

Using `kubectl apply -f` and `kubectl get cwft`

### UI

`ClusterWorkflowTemplate` resources can also be managed by the UI
