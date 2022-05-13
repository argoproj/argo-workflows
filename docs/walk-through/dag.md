# DAG

As an alternative to specifying sequences of steps, you can define the workflow as a directed-acyclic graph (DAG) by specifying the dependencies of each task. This can be simpler to maintain for complex workflows and allows for maximum parallelism when running tasks.

In the following workflow, step `A` runs first, as it has no dependencies. Once `A` has finished, steps `B` and `C` run in parallel. Finally, once `B` and `C` have completed, step `D` can run.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-diamond-
spec:
  entrypoint: diamond
  templates:
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.7
      command: [echo, "{{inputs.parameters.message}}"]
  - name: diamond
    dag:
      tasks:
      - name: A
        template: echo
        arguments:
          parameters: [{name: message, value: A}]
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters: [{name: message, value: B}]
      - name: C
        dependencies: [A]
        template: echo
        arguments:
          parameters: [{name: message, value: C}]
      - name: D
        dependencies: [B, C]
        template: echo
        arguments:
          parameters: [{name: message, value: D}]
```

The dependency graph may have [multiple roots](https://github.com/argoproj/argo-workflows/tree/master/examples/dag-multiroot.yaml). The templates called from a DAG or steps template can themselves be DAG or steps templates. This can allow for complex workflows to be split into manageable pieces.

The DAG logic has a built-in `fail fast` feature to stop scheduling new steps, as soon as it detects that one of the DAG nodes is failed. Then it waits until all DAG nodes are completed before failing the DAG itself.
The [FailFast](https://github.com/argoproj/argo-workflows/tree/master/examples/dag-disable-failFast.yaml) flag default is `true`,  if set to `false`, it will allow a DAG to run all branches of the DAG to completion (either success or failure), regardless of the failed outcomes of branches in the DAG. More info and example about this feature at [here](https://github.com/argoproj/argo-workflows/issues/1442).
