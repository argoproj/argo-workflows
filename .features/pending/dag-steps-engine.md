Description: DAG and Steps templates run on one engine
Authors: [Isitha Subasinghe](https://github.com/isubasinghe)
Component: General
Issues: 12192 14767 16450 16454

DAG and Steps templates are now executed by a single engine, so fixes to readiness, retries, hooks and omission apply to both template types.

A step group whose every step was omitted is now `Omitted` rather than `Succeeded`, so that retrying or resubmitting a workflow that failed in an earlier group works.

An expanded step now has a `TaskGroup` node holding its items, as an expanded DAG task has.
The upgrade notes describe the following behavior changes in detail.

In Steps, a step that fails before it runs errors only itself: its siblings still run, and `continueOn.error` covers it, as in a DAG.
An expanded task or step whose items end both `Error` and `Failed` is `Error`, whatever the item order.
A task after a nested DAG or Steps template sees that template's own phase, not its inner nodes' phases.
A memoized DAG or Steps template whose result cannot be saved to the cache ends `Error`.

A hook that ends `Error`, including one that could not start or timed out, ends its DAG or Steps template `Error` once running tasks finish, unless the template has already failed, and no new task starts meanwhile; `continueOn.error` does not cover it.
A timed-out hook, including a workflow-level hook or the workflow `onExit` node, is recorded as `Error` rather than `Failed`.
A workflow-level hook's expression error is recorded on an `Error` hook node.
Lifecycle hooks run once per item of an expanded DAG task, and not at all for skipped or omitted DAG tasks.
A DAG task's exit hook always waits for its lifecycle hooks.

`{{workflow.failures}}` does not list `TaskGroup` nodes.
After the upgrade, the `TaskGroup` node of an expanded step has no duration estimate until one run succeeds.
A `withSequence` with a negative `count` expands to no items.
A DAG `TaskGroup` node records the resolved `templateRef`.

A step group after a failed, stopped or timed-out group is shown as `Omitted`, and an empty step group after a failed group is `Omitted`.
A step group with a step that ended `Error` is `Error`, and names the step.
A daemon that dies after its group finished fails the template and workflow, while the group stays `Succeeded`.
A daemon that dies this way also runs its exit hook and a lifecycle hook matching its ending phase, and the template waits for them before ending.

`globalName` outputs are exported when their node finishes, so the last to finish wins, and memoized, HTTP, plugin and resumed suspend steps export them at once.
An entrypoint DAG template's own `globalName` outputs are now exported to `workflow.status.outputs`, as for Steps.
The outputs of a DAG template can refer to `workflow.outputs`.
Template metrics are emitted for memoize cache hits.
A DAG task's mutex or semaphore lock, its completion metrics and its `globalName` export happen when the task finishes, even while its lifecycle hook still runs; dependants and the template still wait for the hook.
A failed DAG template's node names its failed task (`child '<node ID>' failed`), which `lastRetry.message` sees.
A DAG task node's error message no longer starts with `task '<node name>' errored:`, and the workflow message for an entry template's own error no longer starts with `error in entry template execution:`.
In a `containerSet` whose pod is deleted, containers that had finished keep their phase.

`argo retry` re-runs a failed DAG task whose dependants were all omitted.
`argo retry` completes, without a new attempt, a succeeded task's or step's exit hook that has its own retry strategy.
After rolling the controller back, delete or terminate running Steps workflows that have an expanded step, and resubmit them.
