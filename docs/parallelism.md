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

When namespace parallelism is enabled, it is plausible for a workflow with a lower priority to be run first if a namespace is at its namespace parallelism limits.

!!! Note
    Workflows that are executing but restricted from running more nodes due to other mechanisms will still count toward parallelism limits.

In addition to the default parallelism, you are able to set individual limits on namespace parallelism by modifying the namespace object with a `workflows.argoproj.io/parallelism-limit` label. Note that individual limits on namespaces will override global namespace limits.

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
