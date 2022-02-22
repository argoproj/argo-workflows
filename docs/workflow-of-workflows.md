# Workflow of Workflows

> v2.9 and after

## Introduction
The Workflow of Workflows pattern involves a parent workflow triggering one or more child workflows, managing them, and acting their results.
 
## Examples
You can use `workflowTemplateRef` to trigger a workflow inline.  
1. Define your workflow as a `workflowtemplate`.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-submittable
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
2. Create the `Workflowtemplate` in cluster using `argo template create <yaml>`
3. Define the workflow of workflows. 
```yaml
# This template demonstrates a workflow of workflows.
# Workflow triggers one or more workflow and manage it.
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-of-workflows-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: workflow1
            template: triggerWorkflowUsingResourceWithoutArgument
            arguments:
              parameters:
              - name: workflowtemplate
                value: "workflow-template-submittable"
        - - name: workflow2
            template: triggerWorkflowUsingResourceWithArgument
            arguments:
              parameters:
              - name: workflowtemplate
                value: "workflow-template-submittable"
              - name: message
                value: "Welcome Argo"

    - name: triggerWorkflowUsingResourceWithoutArgument
      inputs:
        parameters:
          - name: workflowtemplate
      resource:
        action: create
        manifest: |
          apiVersion: argoproj.io/v1alpha1
          kind: Workflow
          metadata:
            generateName: workflow-of-workflows-1-
          spec:
            workflowTemplateRef:
              name: {{inputs.parameters.workflowtemplate}}
        successCondition: status.phase == Succeeded
        failureCondition: status.phase in (Failed, Error)

    - name: triggerWorkflowUsingResourceWithArgument
      inputs:
        parameters:
          - name: workflowtemplate
          - name: message
      resource:
        action: create
        manifest: |
          apiVersion: argoproj.io/v1alpha1
          kind: Workflow
          metadata:
            generateName: workflow-of-workflows-2-
          spec:
            arguments:
              parameters:
              - name: message
                value: {{inputs.parameters.message}}
            workflowTemplateRef:
              name: {{inputs.parameters.workflowtemplate}}
        successCondition: status.phase == Succeeded
        failureCondition: status.phase in (Failed, Error)
```
