# Lifecycle-Hook

## Introduction

A [`LifecycleHook`](fields.md#lifecyclehook) triggers an action based on a conditional expression or on completion of a step or template.
It is configured either at the workflow-level or template-level, for instance as a function of the `workflow.status` or `steps.status`, respectively.
A `LifecycleHook` executes during execution time, at most once for the workflow, step or task it is attached to.
An expanded step or DAG task (one with `withItems`, `withParam` or `withSequence`) runs its hooks once for each item.
So a hook's arguments can use `{{item}}`.
The hook runs in parallel to its step or template once the expression is satisfied.

In other words, a `LifecycleHook` functions like an [exit handler](https://github.com/argoproj/argo-workflows/blob/main/examples/exit-handlers.yaml) with a conditional expression. You must not name a `LifecycleHook` `exit` or it becomes an exit handler; otherwise the hook name has no relevance.

A step's or task's exit hook starts only after any of its lifecycle hooks that started have finished, and the steps or tasks that depend on it wait for all of its hooks.
A step's or task's hook that ends `Error`, for example because it could not be started or it timed out while still `Pending`, fails its DAG or Steps template, and `continueOn` does not cover it.
A hook that ends `Failed` is ignored.
[When a hook ends `Error`](#when-a-hook-ends-error) describes this in detail.

**Workflow-level `LifecycleHook`**: Executes the template when a configured expression is met during the workflow.

- [Workflow-level Lifecycle-Hook example](https://github.com/argoproj/argo-workflows/blob/main/examples/life-cycle-hooks-wf-level.yaml)

**Template-level `Lifecycle-Hook`**: Executes the template when a configured expression is met during the step in which it is defined.

- [Template-level Lifecycle-Hook example](https://github.com/argoproj/argo-workflows/blob/main/examples/life-cycle-hooks-tmpl-level.yaml)

## When hooks run

These rules apply to the hooks of a step or task, in both DAG and Steps templates.

A step or task that did not run does not run its lifecycle hooks or its exit hook.
A step or task does not run when its `when` clause is false, or when its dependencies omit it.

A step's or task's lifecycle hooks always run before its exit hook.
This includes a step or task that finishes in the same reconciliation that starts it, for example a memoization cache hit.
Its exit hook, and the steps or tasks that depend on it, wait for the lifecycle hooks to finish.

A step or task with a `retryStrategy` can have an attempt that succeeds while one of its lifecycle hooks is still running.
The controller does not start another attempt.

A step or task releases its mutex or semaphore lock when it finishes, even if its lifecycle hooks or exit hook are still running.
So a hook can use the same lock, for example through `templateDefaults.synchronization`.
The controller also exports the step's or task's completion metrics and `globalName` outputs when the step or task finishes.
The steps or tasks that depend on it, and its DAG or Steps template, still wait for the hooks to finish.

## When a hook ends `Error`

A hook of a step or task can end `Error` in these cases:

- The hook could not start, for example because an admission webhook denied its pod, or because its expression could not be evaluated.
- The hook was still `Pending` when the hook template's `timeout` or `pendingTimeout` passed.
- The hook errored while it ran, for example because its pod was deleted.

The hook error fails the DAG or Steps template.
`continueOn.error` on the step or task does not cover an error from its hooks.

After a hook error:

- No step or task that has not started begins.
  A step or task that has not started gets no node, so `argo retry` runs it.
- An expanded step or task (with `withItems`, `withParam` or `withSequence`) that has already started creates no more of the items that `parallelism` held back.
  Its `TaskGroup` node completes from the items it has created.
  If it has created no items, it ends `Omitted` with the message `items not started because a hook errored: <count>`.
  `argo retry` creates the remaining items.
- A step or task that is being retried continues to make attempts.
- Steps and tasks that are already running finish.
  Then the template ends `Error` with the hook's message.

A lifecycle hook can error while its step or task is still running.
In this case, the controller marks the step or task `Error` at once, and the template does not wait for its pod to finish.
The template then fails as it does for any step or task that ended `Error`:

- A DAG template ends `Error` with the message `child '<node ID>' failed`.
- A Steps template ends `Failed` with the message of its step group.

The template can still be waiting for a sibling step or task that had already started.
The pod of the step or task whose hook errored can finish during this time.
Then the pod's phase replaces the `Error`, and the template ends `Error` with the hook's message.
So the siblings that start, and the final phase and message of the template, can depend on the order the steps are listed in.
In a DAG template, they can depend on the task names, and on the order of `target` when it is set.

If another step or task had already failed or errored, the template ends with that failure's phase and message instead.

A hook that runs and ends `Failed` is ignored.

### Workflow-level hooks and `onExit`

A workflow-level hook (`spec.hooks`) or a workflow `onExit` node that is still `Pending` when its `timeout` or `pendingTimeout` passes ends `Error`.
This does not change the workflow's own outcome.

When the expression of a workflow-level hook cannot be evaluated, the controller records the error on an `Error` hook node named `<workflow name>.hooks.<hook name>`.
If the workflow's entry node has already finished, the entry node keeps its phase and the workflow's phase does not change.

## Supported conditions

- [Exit handler variables](variables.md#exit-handler): `workflow.status` and `workflow.failures`
- [`template`](fields.md#template)
- [`templateRef`](fields.md#templateref)
- [`arguments`](https://github.com/argoproj/argo-workflows/blob/main/examples/conditionals.yaml)

Hook arguments and expressions that reference outputs of a skipped or omitted step resolve according to [Outputs of Skipped and Omitted Nodes](variables.md#outputs-of-skipped-and-omitted-nodes).

## Unsupported conditions

- [`outputs`](fields.md#outputs) are not usable since `LifecycleHook` executes during execution time and `outputs` are not produced until the step is completed. You can use outputs from previous steps, just not the one you're hooking into. If you'd like to use outputs create an exit handler instead - all the status variable are available there so you can still conditionally decide what to do.

## Notification use case

A `LifecycleHook` can be used to configure a notification depending on a workflow status change or template status change, like the example below:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 generateName: lifecycle-hook-
spec:
 entrypoint: main
 hooks:
   exit:
     template: http
   running:
     expression: workflow.status == "Running"
     template: http
 templates:
   - name: main
     steps:
       - - name: step1
           template: heads

   - name: heads
     container:
       image: alpine:3.23
       command: [sh, -c]
       args: ["echo \"it was heads\""]

   - name: http
     http:
       url: http://dummy.restapiexample.com/api/v1/employees
```

> Put differently, an exit handler is like a workflow-level `LifecycleHook` with an expression of `workflow.status == "Succeeded"` or `workflow.status == "Failed"` or `workflow.status == "Error"`.
