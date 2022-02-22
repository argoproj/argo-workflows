# Core Concepts

This page serves as an introduction into the core concepts of Argo.

## The `Workflow`

The [`Workflow`](fields.md#workflow) is the most important resource in Argo and serves two important functions:

1. It defines the workflow to be executed.
1. It stores the state of the workflow.

Because of these dual responsibilities, a `Workflow` should be treated as a "live" object. It is not only a static definition, but is also an "instance" of said definition. (If it isn't clear what this means, it will be explained below).

### Workflow Spec

The workflow to be executed is defined in the [`Workflow.spec`](fields.md#workflowspec) field. The core structure of a Workflow spec is a list of [`templates`](fields.md#template) and an `entrypoint`.

[`templates`](fields.md#template) can be loosely thought of as "functions": they define instructions to be executed.
The `entrypoint` field defines what the "main" function will be – that is, the template that will be executed first.

Here is an example of a simple `Workflow` spec with a single `template`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-  # Name of this Workflow
spec:
  entrypoint: whalesay        # Defines "whalesay" as the "main" template
  templates:
  - name: whalesay            # Defining the "whalesay" template
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]   # This template runs "cowsay" in the "whalesay" image with arguments "hello world"
```

### `template` Types

There are 6 types of templates, divided into two different categories.

#### Template Definitions

These templates _define_ work to be done, usually in a Container.

##### [Container](fields.md#container)

Perhaps the most common template type, it will schedule a Container. The spec of the template is the same as the [K8s container spec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#container-v1-core), so you can define a container here the same way you do anywhere else in K8s.
    
Example:
```yaml
  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
```
  
##### [Script](fields.md#scripttemplate)

A convenience wrapper around a `container`. The spec is the same as a container, but adds the `source:` field which allows you to define a script in-place.
The script will be saved into a file and executed for you. The result of the script is automatically exported into an [Argo variable](./variables.md) either `{{tasks.<NAME>.outputs.result}}` or `{{steps.<NAME>.outputs.result}}`, depending how it was called. 
    
Example:
```yaml
  - name: gen-random-int
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        import random
        i = random.randint(1, 100)
        print(i)
```

##### [Resource](fields.md#resourcetemplate)

Performs operations on cluster Resources directly. It can be used to get, create, apply, delete, replace, or patch resources on your cluster.
    
This example creates a `ConfigMap` resource on the cluster:
```yaml
  - name: k8s-owner-reference
    resource:
      action: create
      manifest: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          generateName: owned-eg-
        data:
          some: value
```
  
##### [Suspend](fields.md#suspendtemplate)

A suspend template will suspend execution, either for a duration or until it is resumed manually. Suspend templates can be resumed from the CLI (with `argo resume`), the API endpoint<!-- TODO: LINK -->, or the UI.
        
Example:
```yaml
  - name: delay
    suspend:
      duration: "20s"
```
  
#### Template Invocators

These templates are used to invoke/call other templates and provide execution control.

##### [Steps](fields.md#workflowstep)

A steps template allows you to define your tasks in a series of steps. The structure of the template is a "list of lists". Outer lists will run sequentially and inner lists will run in parallel. If you want to run inner lists one by one, use the [Synchronization](fields.md#synchronization) feature. You can set a wide array of options to control execution, such as [`when:` clauses to conditionally execute a step](https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/coinflip.yaml).
    
In this example `step1` runs first. Once it is completed, `step2a` and `step2b` will run in parallel:
```yaml
  - name: hello-hello-hello
    steps:
    - - name: step1
        template: prepare-data
    - - name: step2a
        template: run-data-first-half
      - name: step2b
        template: run-data-second-half
```

##### [DAG](fields.md#dagtemplate)

A dag template allows you to define your tasks as a graph of dependencies. In a DAG, you list all your tasks and set which other tasks must complete before a particular task can begin. Tasks without any dependencies will be run immediately.
    
In this example `A` runs first. Once it is completed, `B` and `C` will run in parallel and once they both complete, `D` will run:
```yaml
  - name: diamond
    dag:
      tasks:
      - name: A
        template: echo
      - name: B
        dependencies: [A]
        template: echo
      - name: C
        dependencies: [A]
        template: echo
      - name: D
        dependencies: [B, C]
        template: echo
```
