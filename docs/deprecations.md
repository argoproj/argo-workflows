# Deprecations

Sometimes a feature of Argo Workflows is deprecated.
You should stop using a deprecated feature as it may be removed in a future minor or major release of Argo Workflows.

To determine if you are using a deprecated feature the [`deprecated_feature`](metrics.md#deprecated_feature) metric can help.
This metric will go up for each use of a deprecated feature by the workflow controller.
This means it may go up once or many times for a single event.
If the number is going up the feature is still in use by your system.
If the metric is not present or no longer increasing are no longer using the monitored deprecated features.

## Deprecated features

### `workflow spec.shutdown`

> v4.2 and after

Stopping or terminating a running Workflow by setting `spec.shutdown` directly is deprecated in favour of creating a [WorkflowAction](workflow-actions.md).
The `argo` CLI, the Argo Server, the UI and the CronWorkflow `Replace` concurrency policy already use WorkflowActions; this only affects clients that patch the Workflow themselves, and older versions of the CLI and Server.

### `workflow spec.suspend`

> v4.2 and after

Suspending a running Workflow by setting `spec.suspend` directly is deprecated.
Setting `spec.suspend` at creation time, to start a Workflow suspended, is still supported and not counted by the metric; `spec.startSuspended: true` does the same and is preferred.
To suspend or resume a running Workflow, use a [WorkflowAction](workflow-actions.md).

## Removed features

These features are removed in this version of workflows.
The [`convert`](cli/argo_convert.md) command in the CLI can update your Workflows, WorkflowTemplates, ClusterWorkflowTemplate and CronWorkflows to the modern syntax.

### `cronworkflow schedule`

The spec field `schedule` which takes a single value is replaced by `schedules` which takes a list.
To update this replace the `schedule` with `schedules` as in the following example

```yaml
spec:
  schedule: "30 1 * * *"
```

is replaced with

```yaml
spec:
  schedules:
    - "30 1 * * *"
```

### `synchronization mutex`

The synchronization field `mutex` which takes a single value is replaced by `mutexes` which takes a list.
To update this replace `mutex` with `mutexes` as in the following example

```yaml
synchronization:
  mutex:
    name: foobar
```

is replaced with

```yaml
synchronization:
  mutexes:
    - name: foobar
```

### `synchronization semaphore`

The synchronization field `semaphore` which takes a single value is replaced by `semaphores` which takes a list.
To update this replace `semaphore` with `semaphores` as in the following example

```yaml
synchronization:
  semaphore:
    configMapKeyRef:
      name: my-config
      key: workflow
```

is replaced with

```yaml
synchronization:
  semaphores:
    - configMapKeyRef:
        name: my-config
        key: workflow
```

### `workflow podpriority`

The Workflow spec field `podPriority` which takes a numeric value is deprecated and `podPriorityClassName` should be used instead.
To update this you will need a [PriorityClass](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/#priorityclass) in your cluster and refer to that using `podPriorityClassName`.

This field cannot be updated by the CLI `convert` command.
