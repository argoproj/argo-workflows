# Cluster Workflow Templates

> v2.8 and after

## Introduction

`ClusterWorkflowTemplates` are cluster scoped `WorkflowTemplates`. `ClusterWorkflowTemplate`
can be created cluster scoped like `ClusterRole` and can be accessed across all namespaces in the cluster.

`WorkflowTemplates` documentation [link](./workflow-templates.md)

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
```

## Referencing other `ClusterWorkflowTemplates`

You can reference `templates` from other `ClusterWorkflowTemplates` using a `templateRef` field with `clusterScope: true` .
Just as how you reference other `templates` within the same `Workflow`, you should do so from a `steps` or `dag` template.

Here is an example:

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
            name: cluster-workflow-template-whalesay-template   # This is the name of the "WorkflowTemplate or ClusterWorkflowTemplate" CRD that contains the "template" you want
            template: whalesay-template # This is the name of the "template" you want to reference
            clusterScope: true          # This field indicates this templateRef is pointing ClusterWorkflowTemplate
          arguments:                    # You can pass in arguments as normal
            parameters:
            - name: message
              value: "hello world"
```

> 2.9 and after

### Create `Workflow` from `ClusterWorkflowTemplate` Spec

You can create `Workflow` from `ClusterWorkflowTemplate` spec using `workflowTemplateRef` with `clusterScope: true`. If you pass the arguments to created `Workflow`, it will be merged with cluster workflow template arguments

Here is an example for `ClusterWorkflowTemplate` with `entrypoint` and `arguments`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-workflow-template-submittable
spec:
  entrypoint: whalesay-template
  arguments:
    parameters:
      - name: message
        value: hello world
  templates:
    - name: whalesay-template
      inputs:
        parameters:
          - name: message
      container:
        image: docker/whalesay
        command: [cowsay]
        args: ["{{inputs.parameters.message}}"]

```

Here is an example for creating `ClusterWorkflowTemplate` as Workflow with passing `entrypoint` and `arguments` to `ClusterWorkflowTemplate`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: cluster-workflow-template-hello-world-
spec:
  entrypoint: whalesay-template
  arguments:
    parameters:
      - name: message
        value: "from workflow"
  workflowTemplateRef:
    name: cluster-workflow-template-submittable
    clusterScope: true
```  

Here is an example of a creating `WorkflowTemplate` as Workflow and using `WorkflowTemplates`'s `entrypoint` and `Workflow Arguments`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: cluster-workflow-template-hello-world-
spec:
  workflowTemplateRef:
    name: cluster-workflow-template-submittable
    clusterScope: true

```

## Managing `ClusterWorkflowTemplates`

### CLI

You can create some example templates as follows:

```bash
argo cluster-template create https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/cluster-workflow-template/clustertemplates.yaml
```

The submit a workflow using one of those templates:

```bash
argo submit https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/cluster-workflow-template/cluster-wftmpl-dag.yaml
```

> 2.7 and after
>
The submit a `ClusterWorkflowTemplate` as a `Workflow`:

```bash
argo submit --from clusterworkflowtemplate/workflow-template-submittable
```

### `kubectl`

Using `kubectl apply -f` and `kubectl get cwft`

### UI

`ClusterWorkflowTemplate` resources can also be managed by the UI
