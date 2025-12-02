# DAG

As an alternative to specifying sequences of [steps](steps.md), you can define a workflow as a directed-acyclic graph (DAG) by specifying the dependencies of each task.
DAGs can be simpler to maintain for complex workflows and allow for maximum parallelism when running tasks.

In the following workflow, step `A` runs first, as it has no dependencies.
Once `A` has finished, steps `B` and `C` run in parallel.
Finally, once `B` and `C` have completed, step `D` runs.

/// tab | YAML

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

///

/// tab | Python

```python
from hera.workflows import DAG, Container, Parameter, Workflow # (1)!

with Workflow(generate_name="dag-diamond-", entrypoint="diamond") as w:
    echo = Container(
        name="echo",
        image="alpine:3.7",
        command=["echo", "{{inputs.parameters.message}}"],
        inputs=[Parameter(name="message")],
    )
    with DAG(name="diamond"):
        A = echo(name="A", arguments={"message": "A"})
        B = echo(name="B", arguments={"message": "B"})
        C = echo(name="C", arguments={"message": "C"})
        D = echo(name="D", arguments={"message": "D"})
        A >> [B, C] >> D # (2)!
```

1. Install the `hera` package to define your Workflows in Python. Learn more at [the Hera docs](https://hera.readthedocs.io/en/stable/).
2. Hera uses [enhanced depends logic](../enhanced-depends-logic.md) when using `>>` to define dependencies.

///

The dependency graph may have [multiple roots](https://github.com/argoproj/argo-workflows/tree/main/examples/dag-multiroot.yaml).
The templates called from a DAG or steps template can themselves be DAG or steps templates, allowing complex workflows to be split into manageable pieces.

## Enhanced Depends

For more complicated, conditional dependencies, you can use the [Enhanced Depends](../enhanced-depends-logic.md) feature.

## Fail Fast

By default, DAGs fail fast: when one task fails, no new tasks will be scheduled.
Once all running tasks are completed, the DAG will be marked as failed.

If [`failFast`](https://github.com/argoproj/argo-workflows/tree/main/examples/dag-disable-failFast.yaml) is set to `false` for a DAG, all branches will run to completion, regardless of failures in other branches.
