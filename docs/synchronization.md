# Synchronization

> v2.10 and after

You can use synchronization to limit the parallel execution of workflows or templates.
You can use mutexes to restrict workflows or templates to only having a single concurrent section.
You can use semaphores to restrict workflows or templates to a configured number of parallel runs.
This documentation refers "locks" to mean mutexes and semaphores.

You can create multiple synchronization configurations in the `ConfigMap` that can be referred to from a workflow or template.

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

Each synchronization block may only refer to either a semaphore or a mutex.
If you specify both only the semaphore will be locked.

## Workflow-level Synchronization

You can limit parallel execution of a workflow by using Workflow-level synchronization.
If multiple workflows have the same synchronization reference they will be limited by that synchronization reference.

In this example, Workflow refers to `workflow` synchronization key which is configured as limit `"1"`, so only one workflow instance will be executed at given time even if multiple workflows are created.

Using a semaphore configured by a `ConfigMap`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-wf-level-
spec:
  entrypoint: hello-world
  synchronization:
    semaphore:
      configMapKeyRef:
        name: my-config
        key: workflow
  templates:
  - name: hello-world
    container:
      image: busybox
      command: [echo]
      args: ["hello world"]
```

Using a mutex achieves the same thing as a count `"1"` semaphore:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-wf-level-
spec:
  entrypoint: hello-world
  synchronization:
    mutex:
      name: workflow
  templates:
  - name: hello-world
    container:
      image: busybox
      command: [echo]
      args: ["hello world"]
```

## Template-level Synchronization

You can limit parallel execution of a template by using Template-level synchronization.
If templates have the same synchronization reference they will be limited by that synchronization reference, across all workflows.

In this example, `acquire-lock` template has synchronization reference of `template` key which is configured as limit `"2"` so a maximum of two instances of the `acquire-lock` template will be executed at a given time.
This applies even multiple steps or tasks within a workflow or different workflows refer to the same template.

Using a semaphore configured by a `ConfigMap`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-tmpl-level-
spec:
  entrypoint: synchronization-tmpl-level-example
  templates:
  - name: synchronization-tmpl-level-example
    steps:
    - - name: synchronization-acquire-lock
        template: acquire-lock
        arguments:
          parameters:
          - name: seconds
            value: "{{item}}"
        withParam: '["1","2","3","4","5"]'

  - name: acquire-lock
    synchronization:
      semaphore:
        configMapKeyRef:
          name: my-config
          key: template
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["sleep 10; echo acquired lock"]
```

Using a mutex will limit to a single execution of the template at any one time:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-tmpl-level-
spec:
  entrypoint: synchronization-tmpl-level-example
  templates:
  - name: synchronization-tmpl-level-example
    steps:
    - - name: synchronization-acquire-lock
        template: acquire-lock
        arguments:
          parameters:
          - name: seconds
            value: "{{item}}"
        withParam: '["1","2","3","4","5"]'

  - name: acquire-lock
    synchronization:
      mutex:
        name: template
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["sleep 10; echo acquired lock"]
```

Examples:

1. [Workflow level semaphore](https://github.com/argoproj/argo-workflows/blob/main/examples/synchronization-wf-level.yaml)
1. [Workflow level mutex](https://github.com/argoproj/argo-workflows/blob/main/examples/synchronization-mutex-wf-level.yaml)
1. [Step level semaphore](https://github.com/argoproj/argo-workflows/blob/main/examples/synchronization-tmpl-level.yaml)
1. [Step level mutex](https://github.com/argoproj/argo-workflows/blob/main/examples/synchronization-mutex-tmpl-level.yaml)

## Queuing

When a Workflow cannot take a lock it will be placed into a ordered queue.

Workflows can have a `priority` set in their specification.
The queue is first ordered by priority, with a higher priority number being placed before a lower priority number.
The queue is then ordered by `CreationTimestamp` of the Workflow; older Workflows will be ordered before newer workflows.

Workflows are only be allowed to take a lock if they are at the front of the queue for that lock.

## Other Parallelism support

You can also [restrict parallelism in other ways](parallelism.md).
