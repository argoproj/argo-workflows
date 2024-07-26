# Limiting parallelism

You can restrict the number of simultaneous workflow executions.

## Controller level

You can limit the total number of workflows that can execute at any one time in the [workflow controller ConfigMap](./workflow-controller-configmap.yaml).

```yaml
data:
  parallelism: "10"
```

You can also limit the number of workflows that can execute in a single namespace.

```yaml
data:
  namespaceParallelism: "4"
```

Workflows that are executing but restricted from running more nodes due to the other mechanisms will still count towards the parallelism limits.

### Priority

Workflows can have a `priority` set in their specification.

Workflows with a higher priority number that have not started due to controller level parallelism will be started before lower priority workflows.

## Workflow level

You can restrict parallelism within a workflow using `parallelism` within a workflow or template.
This only restricts total concurrent executions of steps or tasks within the same workflow.

Examples:

1. [`parallelism-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-limit.yaml) restricts the parallelism of a [loop](./walk-through/loops.md)
1. [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-nested.yaml) restricts the parallelism of a nested loop
1. [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-nested-dag.yaml) restricts the number of dag tasks that can be run at any one time
1. [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-nested-workflow.yaml) shows how parallelism is inherited by children
1. [`parallelism-template-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-template-limit.yaml) shows how parallelism of looped templates is also restricted

## Synchronization

You can use [mutexes and semaphores](./synchronization.md) to control the parallel execution of sections of a workflow.
