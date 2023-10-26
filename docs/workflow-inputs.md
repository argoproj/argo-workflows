# Workflow Inputs

## Introduction

`Workflows` and `template`s operate on a set of defined parameters and arguments that are supplied to the running container. The precise details of how to manage the inputs can be confusing; this article attempts to clarify concepts and provide simple working examples to illustrate the various configuration options.

The examples below are limited to `DAGTemplate`s and mainly focused on `parameters`, but similar reasoning applies to the other types of `template`s.

### Parameter Inputs

First, some clarification of terms is needed. For a glossary reference, see [Argo Core Concepts](workflow-concepts.md).

A `workflow` provides `arguments`, which are passed in to the entry point template. A `template` defines `inputs` which are then provided by template callers (such as `steps`, `dag`, or even a `workflow`). The structure of both is identical.

For example, in a `Workflow`, one parameter would look like this:

```yaml
arguments:
  parameters:
  - name: workflow-param-1
```

And in a `template`:

```yaml
inputs:
  parameters:
  - name: template-param-1
```

Inputs to `DAGTemplate`s use the `arguments` format:

```yaml
dag:
  tasks:
  - name: step-A
    template: step-template-a
    arguments:
      parameters:
      - name: template-param-1
        value: abcd
```

Previous examples in context:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: example-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: workflow-param-1
  templates:
  - name: main
    dag:
      tasks:
      - name: step-A 
        template: step-template-a
        arguments:
          parameters:
          - name: template-param-1
            value: "{{workflow.parameters.workflow-param-1}}"
 
  - name: step-template-a
    inputs:
      parameters:
        - name: template-param-1
    script:
      image: alpine
      command: [/bin/sh]
      source: |
          echo "{{inputs.parameters.template-param-1}}"
```

To run this example: `argo submit -n argo example.yaml -p 'workflow-param-1="abcd"' --watch`

### Using Previous Step Outputs As Inputs

In `DAGTemplate`s, it is common to want to take the output of one step and send it as the input to another step. However, there is a difference in how this works for artifacts vs parameters. Suppose our `step-template-a` defines some outputs:

```yaml
outputs:
  parameters:
    - name: output-param-1
      valueFrom:
        path: /p1.txt
  artifacts:
    - name: output-artifact-1
      path: /some-directory
```

In my `DAGTemplate`, I can send these outputs to another template like this:

```yaml
dag:
  tasks:
  - name: step-A 
    template: step-template-a
    arguments:
      parameters:
      - name: template-param-1
        value: "{{workflow.parameters.workflow-param-1}}"
  - name: step-B
    dependencies: [step-A]
    template: step-template-b
    arguments:
      parameters:
      - name: template-param-2
        value: "{{tasks.step-A.outputs.parameters.output-param-1}}"
      artifacts:
      - name: input-artifact-1
        from: "{{tasks.step-A.outputs.artifacts.output-artifact-1}}"
```

Note the important distinction between `parameters` and `artifacts`; they both share the `name` field, but one uses `value` and the other uses `from`.
