# Cluster Workflow Templates

> v2.8 and after

## Introduction

`ClusterWorkflowTemplates` are cluster scoped `WorkflowTemplates`. `ClusterWorkflowTemplate` 
can be created cluster scoped like `ClusterRole` and can be accessed all namespaces in the cluster. 

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

You can reference `templates` from another `ClusterWorkflowTemplates` using a `templateRef` field with `clusterScope: true` .
Just as how you reference other `templates` within the same `Workflow`, you should do so from a `steps` or `dag` template.

Here is an example:
More examples []()
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
#### Referring `ClusterWorkflowTemplate` as Workflow
You can refer the `ClusterWorkflowTemplate` as `workflow` without defining templates. If `Workflow` has `arguments` that will be merged with `ClusterWorkflowTemplate` arguments and Workflow argument value will get overwrite with ClusterWorkflowTemplate argument value.

Here is an example of a referring `WorkflowTemplate` as Workflow with passing `entrypoint` and  `arguments` to `ClusterWorkflowTemplate`
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  entrypoint: whalesay-template
  arguments:
    parameters:
      - name: message
        value: "from workflow"  # This will be passed to ClusterWorkflowTemplate argument
  workflowTemplateRef:
    name: workflow-template-submittable
    clusterScope: true
```  
Here is an example of a referring `ClusterWorkflowTemplate` as Workflow and using `ClusterWorkflowTemplates`'s `entrypoint` and `arguments`
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  workflowTemplateRef:
    name: workflow-template-submittable
    clusterScope: true
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

> 2.7 and after
>
The submit a `ClusterWorkflowTemplate` as a `Workflow`:
```shell script
argo submit --from clusterworkflowtemplate/workflow-template-submittable
```

### `kubectl`

Using `kubectl apply -f` and `kubectl get cwft`

### UI

`ClusterWorkflowTemplate` resources can also be managed by the UI
