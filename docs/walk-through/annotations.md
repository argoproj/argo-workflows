# Annotations

Argo Workflows now supports annotations as a new field in workflow templates. 

## Adding Annotations to a template

To add annotations to a workflow template, include the `annotations` field in template definition, for example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
    name: example-workflow-template
spec:
    entrypoint: whalesay
    templates:
    - name: whalesay
      annotations:
        workflows.argoproj.io/display-name: "my-custom-display-name"
      container:
          image: docker/whalesay
          command: [cowsay]
          args: ["hello world"]
```

In this example, the annotation `workflows.argoproj.io/display-name` is used to change the node name in the UI to "my-custom-display-name".

## Annotation Templates

Annotations can also be created dynamically using parameters. This allows you to dynamically set annotation values based on input parameters.

Here is an example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: templated-annotations-workflow
spec:
  entrypoint: whalesay
  arguments:
    parameters:
      - name: display-name
        value: "default-display-name"
  templates:
  - name: whalesay
    annotations:
      workflows.argoproj.io/display-name: "{{inputs.parameters.display-name}}"
    inputs:
      parameters:
        - name: display-name
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
```

In this example, the annotation `workflows.argoproj.io/display-name` is set using the `display-name` parameter. You can override this parameter when submitting the workflow to dynamically change the annotation value.

## Supported Annotation Types

Here is a table of all supported annotation types in Argo Workflows:

| Annotation Key                               | Description                                                                 |
|----------------------------------------------|-----------------------------------------------------------------------------|
| `workflows.argoproj.io/display-name`         | Changes the node name in the UI.                                            |
