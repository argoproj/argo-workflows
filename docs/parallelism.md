# Limiting parallelism

You can restrict the number of parallel workflow executions.

## Controller-level

You can limit the total number of parallel workflow executions in the [workflow controller ConfigMap](workflow-controller-configmap.yaml):

```yaml
data:
  parallelism: "10"
```

You can also limit the total number of parallel workflow executions in a single namespace:

```yaml
data:
  namespaceParallelism: "4"
```

!!! Note
    Workflows that are executing but restricted from running more nodes due to other mechanisms will still count toward parallelism limits.

### Priority

You can set a `priority` on workflows:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: priority-
spec:
  priority: 3
  # ...
```

Workflows that have not started due to Controller-level parallelism will be queued: workflows with higher priority numbers will start before lower priority ones.
The default is `priority: 0`.

## Synchronization

You can also use [mutexes, semaphores, and parallelism](synchronization.md) to control the parallel execution of workflows and templates.
