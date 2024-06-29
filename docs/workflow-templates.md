# Workflow Templates

> v2.4 and after

## Introduction

`WorkflowTemplates` are definitions of `Workflows` that live in your cluster. This allows you to create a library of
frequently-used templates and reuse them either by submitting them directly (v2.7 and after) or by referencing them from
your `Workflows`.

### `WorkflowTemplate` vs `template`

The terms `WorkflowTemplate` and `template` have created an unfortunate naming collision and have created some confusion
in the past. However, a quick description should clarify each and their differences.

- A `template` (lower-case) is a task within a `Workflow` or (confusingly) a `WorkflowTemplate` under the field `templates`. Whenever you define a
`Workflow`, you must define at least one (but usually more than one) `template` to run. This `template` can be of type
`container`, `script`, `dag`, `steps`, `resource`, or `suspend` and can be referenced by an `entrypoint` or by other
`dag`, and `step` templates.

Here is an example of a `Workflow` with two `templates`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-
spec:
  entrypoint: hello           # We reference our first "template" here

  templates:
  - name: hello               # The first "template" in this Workflow, it is referenced by "entrypoint"
    steps:                    # The type of this "template" is "steps"
    - - name: hello
        template: whalesay    # We reference our second "template" here
        arguments:
          parameters: [{name: message, value: "hello1"}]

  - name: whalesay             # The second "template" in this Workflow, it is referenced by "hello"
    inputs:
      parameters:
      - name: message
    container:                # The type of this "template" is "container"
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
```

- A `WorkflowTemplate` is a definition of a `Workflow` that lives in your cluster. Since it is a definition of a `Workflow`
it also contains `templates`. These `templates` can be referenced from within the `WorkflowTemplate` and from other `Workflows`
and `WorkflowTemplates` on your cluster. To see how, please see [Referencing Other `WorkflowTemplates`](#referencing-other-workflowtemplates).

## `WorkflowTemplate` Spec

> v2.7 and after

In v2.7 and after, all the fields in `WorkflowSpec` (except for `priority` that must be configured in a `WorkflowSpec` itself) are supported for `WorkflowTemplates`. You can take any existing `Workflow` you may have and convert it to a `WorkflowTemplate` by substituting `kind: Workflow` to `kind: WorkflowTemplate`.

> v2.4 – 2.6

`WorkflowTemplates` in v2.4 - v2.6 are only partial `Workflow` definitions and only support the `templates` and
`arguments` field.

This would **not** be a valid `WorkflowTemplate` in v2.4 - v2.6 (notice `entrypoint` field):

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-submittable
spec:
  entrypoint: whalesay-template     # Fields other than "arguments" and "templates" not supported in v2.4 - v2.6
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

However, this would be a valid `WorkflowTemplate`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-submittable
spec:
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

### Adding labels/annotations to Workflows with `workflowMetadata`

> v2.10.2 and after

To automatically add labels and/or annotations to Workflows created from `WorkflowTemplates`, use `workflowMetadata`.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-submittable
spec:
  workflowMetadata:
    labels:
      example-label: example-value
```

### Working with parameters

When working with parameters in a `WorkflowTemplate`, please note the following:

- When working with global parameters, you can instantiate your global variables in your `Workflow`
and then directly reference them in your `WorkflowTemplate`. Below is a working example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: hello-world-template-global-arg
spec:
  serviceAccountName: argo
  templates:
    - name: hello-world
      container:
        image: docker/whalesay
        command: [cowsay]
        args: ["{{workflow.parameters.global-parameter}}"]
---
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-wf-global-arg-
spec:
  serviceAccountName: argo
  entrypoint: whalesay
  arguments:
    parameters:
      - name: global-parameter
        value: hello
  templates:
    - name: whalesay
      steps:
        - - name: hello-world
            templateRef:
              name: hello-world-template-global-arg
              template: hello-world
```

- When working with local parameters, the values of local parameters must be supplied at the template definition inside
the `WorkflowTemplate`. Below is a working example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: hello-world-template-local-arg
spec:
  templates:
    - name: hello-world
      inputs:
        parameters:
          - name: msg
            value: "hello world"
      container:
        image: docker/whalesay
        command: [cowsay]
        args: ["{{inputs.parameters.msg}}"]
---
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-local-arg-
spec:
  entrypoint: whalesay
  templates:
    - name: whalesay
      steps:
        - - name: hello-world
            templateRef:
              name: hello-world-template-local-arg
              template: hello-world
```

## Referencing other `WorkflowTemplates`

You can reference `templates` from another `WorkflowTemplates` (see the [difference between the two](#workflowtemplate-vs-template)) using a `templateRef` field.
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
          templateRef:                  # You can reference a "template" from another "WorkflowTemplate" using this field
            name: workflow-template-1   # This is the name of the "WorkflowTemplate" CRD that contains the "template" you want
            template: whalesay-template # This is the name of the "template" you want to reference
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
          arguments:
            parameters:
            - name: message
              value: "hello world"
```

You should **never** reference another template directly on a `template` object (outside of a `steps` or `dag` template).
This includes both using `template` and `templateRef`.
This behavior is deprecated, no longer supported, and will be removed in a future version.

Here is an example of a **deprecated** reference that **should not be used**:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    template:                     # You should NEVER use "template" here. Use it under a "steps" or "dag" template (see above).
    templateRef:                  # You should NEVER use "templateRef" here. Use it under a "steps" or "dag" template (see above).
      name: workflow-template-1
      template: whalesay-template
    arguments:                    # Arguments here are ignored. Use them under a "steps" or "dag" template (see above).
      parameters:
      - name: message
        value: "hello world"
```

The reasoning for deprecating this behavior is that a `template` is a "definition": it defines inputs and things to be
done once instantiated. With this deprecated behavior, the same template object is allowed to be an "instantiator":
to pass in "live" arguments and reference other templates (those other templates may be "definitions" or "instantiators").

This behavior has been problematic and dangerous. It causes confusion and has design inconsistencies.

### Create `Workflow` from `WorkflowTemplate` Spec

> v2.9 and after

You can create `Workflow` from `WorkflowTemplate` spec using `workflowTemplateRef`. If you pass the arguments to created `Workflow`, it will be merged with workflow template arguments.
Here is an example for referring `WorkflowTemplate` as Workflow with passing `entrypoint` and `Workflow Arguments` to `WorkflowTemplate`

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
        value: "from workflow"
  workflowTemplateRef:
    name: workflow-template-submittable
```

Here is an example of a referring `WorkflowTemplate` as Workflow and using `WorkflowTemplates`'s `entrypoint` and `Workflow Arguments`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  workflowTemplateRef:
    name: workflow-template-submittable

```

## Managing `WorkflowTemplates`

### CLI

You can create some example templates as follows:

```bash
argo template create https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/workflow-template/templates.yaml
```

Then submit a workflow using one of those templates:

```bash
argo submit https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/workflow-template/hello-world.yaml
```

> v2.7 and after

Then submit a `WorkflowTemplate` as a `Workflow`:

```bash
argo submit --from workflowtemplate/workflow-template-submittable
```

If you need to submit a `WorkflowTemplate` as a `Workflow` with parameters:

```bash
argo submit --from workflowtemplate/workflow-template-submittable -p message=value1
```

### `kubectl`

Using `kubectl apply -f` and `kubectl get wftmpl`

### GitOps via Argo CD

`WorkflowTemplate` resources can be managed with GitOps by using [Argo CD](https://github.com/argoproj/argo-cd)

### UI

`WorkflowTemplate` resources can also be managed by the UI

Users can specify options under `enum` to enable drop-down list selection when submitting `WorkflowTemplate`s from the UI.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-with-enum-values
spec:
  entrypoint: argosay
  arguments:
    parameters:
      - name: message
        value: one
        enum:
          -   one
          -   two
          -   three
  templates:
    - name: argosay
      inputs:
        parameters:
          - name: message
            value: '{{workflow.parameters.message}}'
      container:
        name: main
        image: 'argoproj/argosay:v2'
        command:
          - /argosay
        args:
          - echo
          - '{{inputs.parameters.message}}'
```
