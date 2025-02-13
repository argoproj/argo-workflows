# Synchronization

> v2.10 and after

You can limit the parallel execution of workflows or templates:

- You can use mutexes to restrict workflows or templates to only having a single concurrent execution.
- You can use semaphores to restrict workflows or templates to a configured number of parallel executions.
- You can use parallelism to restrict concurrent tasks or steps within a single workflow.

The term "locks" on this page means mutexes and semaphores.

You can create multiple semaphore configurations in a `ConfigMap` that can be referred to from a workflow or template.

For example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
 name: my-config
data:
  workflow: "1"  # Only one workflow can run at given time in particular namespace
  template: "2"  # Two instances of template can run at a given time in particular namespace
```

## Workflow-level Synchronization

You can limit parallel execution of workflows by using the same synchronization reference.

In this example the synchronization key `workflow` is configured as limit `"1"`, so only one workflow instance will execute at a time even if multiple workflows are created.

Using a semaphore configured by a `ConfigMap`:

```yaml title="examples/synchronization-wf-level.yaml"
--8<-- "examples/synchronization-wf-level.yaml:12"
```

Using a mutex is equivalent to a limit `"1"` semaphore:

```yaml title="examples/synchronization-mutex-wf-level.yaml"
--8<-- "examples/synchronization-mutex-wf-level.yaml:3"
```

## Template-level Synchronization

You can limit parallel execution of templates by using the same synchronization reference.

In this example the synchronization key `template` is configured as limit `"2"`, so a maximum of two instances of the `acquire-lock` template will execute at a time.
This applies even when multiple steps or tasks within a workflow or different workflows refer to the same template.

Using a semaphore configured by a `ConfigMap`:

```yaml title="examples/synchronization-tmpl-level.yaml"
--8<-- "examples/synchronization-tmpl-level.yaml:11"
```

Using a mutex will limit to a single concurrent execution of the template:

```yaml title="examples/synchronization-mutex-tmpl-level.yaml"
--8<-- "examples/synchronization-mutex-tmpl-level.yaml:3"
```

## Queuing

When a workflow cannot acquire a lock it will be placed into a ordered queue.

You can set a [`priority`](parallelism.md#priority) on workflows.
The queue is first ordered by priority: a higher priority number is placed before a lower priority number.
The queue is then ordered by `creationTimestamp`: older workflows are placed before newer workflows.

Workflows can only acquire a lock if they are at the front of the queue for that lock.

## Multiple locks

> v3.6 and after

You can specify multiple locks in a single workflow or template.

```yaml
synchronization:
  mutexes:
    - name: alpha
    - name: beta
  semaphores:
    - configMapKeyRef:
        key: foo
        name: my-config
    - configMapKeyRef:
        key: bar
        name: my-config
```

The workflow will block until all of these locks are available.

## Workflow-level parallelism

You can use `parallelism` within a workflow or template to restrict the total concurrent executions of steps or tasks.
(Note that this only restricts concurrent executions within the same workflow.)

Examples:

1. [`parallelism-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-limit.yaml) restricts the parallelism of a [loop](walk-through/loops.md)
1. [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-nested.yaml) restricts the parallelism of a nested loop
1. [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-nested-dag.yaml) restricts the number of dag tasks that can be run at any one time
1. [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-nested-workflow.yaml) shows how parallelism is inherited by children
1. [`parallelism-template-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-template-limit.yaml) shows how parallelism of looped templates is also restricted

!!! Warning
    If a Workflow is at the front of the queue and it needs to acquire multiple locks, all other Workflows that also need those same locks will wait. This applies even if the other Workflows only wish to acquire a subset of those locks.

## Other Parallelism support

You can also [restrict parallelism at the Controller-level](parallelism.md).
