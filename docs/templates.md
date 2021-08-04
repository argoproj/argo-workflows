# Templates

See [core concepts](core-concepts.md) for DAG, steps, container templates. 

## Container Set Template

See [container set template](container-set-template.md).

## Inline Templates

![alpha](assets/alpha.svg)

> v3.2 and after

You can inline other templates within DAG and steps.

Examples:

* [DAG](examples/dag-inline-workflow.yaml)
* [Steps](examples/steps-inline-workflow.yaml)

!!! Warning
    You can only inline once. Inlining a DAG within a DAG will not work.
