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

## Synchronization

You can use [mutexes, semaphores and parallelism](./synchronization.md) to control the parallel execution of sections of a workflow.
