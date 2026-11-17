# How DAG and Steps Templates Run

> v4.2 and after

DAG templates and Steps templates run on one shared engine.
Both template types follow the same rules for nodes, errors, `continueOn`, timeouts and phases.

In this page, a "step" is an entry in a Steps template and a "task" is an entry in a DAG template.
A step or task is "expanded" when it has `withItems`, `withParam` or `withSequence`.

## Nodes in a DAG or Steps template

Each step group in a Steps template has its own node.
The step group's children are the steps in that group.

An expanded step or task has a `TaskGroup` node.
The items are children of the `TaskGroup` node.
If the expanded step or task has a `templateRef` that refers to outputs of other steps or tasks, the `TaskGroup` node records the resolved `templateRef`.
When the expansion has no items, the step or task gets a `Skipped` node instead, and that node records the resolved `templateRef`.

The controller creates a step group node when the step group starts.
When a step group fails, stops or times out, the step groups after it do not run.
The controller still shows each of those later step groups, with the phase `Omitted`, together with their steps, which are also `Omitted`.
When `failFast` stops a template, the controller does not create the later step groups.

An empty step group (`- []`) that follows a step group that did not succeed ends `Omitted`.

The outbound nodes of a template (the `outboundNodes` field on its node) are the nodes that come last in it.
When a Steps template fails in an earlier step group, its outbound nodes are the `Omitted` steps of its last step group.
The controller links a step or task that runs after this template under those `Omitted` steps, for example when the template's task has `continueOn.failed`.
The workflow graph also draws it there.
If you have tools that read `outboundNodes`, expect these nodes to be `Omitted`.

## When a step or task fails before it runs

A step or task can fail before it runs.
This happens when:

- Its arguments, its `when` clause, its `withParam` or its items cannot be resolved.
- Its pod is rejected.
- Its `podSpecPatch` cannot be applied.

In this case, the step's or task's node ends `Error`.
The error affects only that step or task.
The other steps in its step group still run.

When the other steps have finished, the step group ends `Error` with the message `step group deemed errored due to child <node name> error: <reason>`.
A step group also ends `Error` with this message when a step ends `Error` while it runs, for example because its pod was deleted.
When a step group ends `Error`, the Steps template ends `Failed` with the step group's message.
A step or task that uses this Steps template as its template sees `Failed`, so its `continueOn.failed` applies, and its `continueOn.error` does not.
If nothing handles the failure, the workflow ends `Failed` with this message.

`continueOn.error` on the step or task lets the template continue, in both DAG and Steps templates.

The node of a DAG task that fails before it runs shows the cause as its message.

## When a step waits for an output that was not reported

A step can use an output of an earlier step as an argument, for example `{{steps.producer.outputs.parameters.out}}`.
The producer step can finish without reporting that output, for example when it failed and `continueOn.failed` let the template continue.
This is not an error.
The step waits for the output, and it has no node while it waits.
The other steps in its step group start.
The next step group waits for the step.
If the producer never reports the output, the step waits forever and the workflow stays `Running`.

An output of a `Skipped` or `Omitted` step or task is different.
See [Outputs of Skipped and Omitted Nodes](variables.md#outputs-of-skipped-and-omitted-nodes).

## When a step or task times out while `Pending`

A step or task can still be `Pending` when its template's `timeout` or `pendingTimeout` passes.
Its node then ends `Failed`.
The timeout affects only that step or task.
The step group waits for the other steps in the group to finish.
Then the step group ends `Failed` with the message `child '<node ID>' failed`.
If the step has `continueOn.failed`, the next step group runs instead.

Because the template waits for the other steps, the workflow's exit handler starts only when those steps have finished.

## How `Error` and `Failed` combine

### Expanded steps and tasks

When the items of an expanded step or task include both `Error` items and `Failed` items, the step or task ends `Error`.
The order of the items does not change this result.

This affects everything that reads the phase of the step or task:

- In `depends`, `task.Errored` is true and `task.Failed` is false.
- `continueOn.error` covers the step or task, and `continueOn.failed` does not.
- `{{workflow.status}}` in an exit handler can be `Error`.

In a Steps template, `continueOn` applies to the expanded step as a whole.
For example, an expanded step with `continueOn.error` continues when one item ended `Failed` and another item ended `Error`.

### DAG templates

When tasks that a DAG's target task depends on fail, the controller omits the target task, and the DAG fails.
The DAG ends `Error` if any of those tasks ended `Error`.
Otherwise, the DAG ends `Failed`.
For example, a target `c` has `depends: "a.Succeeded && b.Succeeded"`.
If `a` ends `Failed` and `b` ends `Error`, the DAG ends `Error`.

This affects everything that reads the phase of the DAG: `{{workflow.status}}` in an exit handler, and the `continueOn` and `depends` of a step or task that uses the DAG as its template.

The node of a failed DAG template has the message `child '<node ID>' failed`.
The node ID is the ID of the first failed task in the `tasks` list, whatever order the tasks failed in.
A `retryStrategy.expression` on the DAG template sees this message in `lastRetry.message`.

### Nested templates

A step or task can use a DAG or Steps template as its template.
A task that follows it sees only the phase of that template's node.
It does not see the phases of the nodes inside the template.
For example, a task inside the nested template fails, but `continueOn` lets the nested template succeed.
Then an `Omitted` or `Skipped` task after the nested template does not fail the outer DAG.

### The workflow message

The entrypoint template can end `Error` because of its own error, for example when its outputs cannot be resolved.
The workflow message is then that error.

## `argo retry`

`argo retry` resets these nodes so that they run again:

- A failed DAG task whose dependent tasks were all `Omitted`, for example because its `when` clause or `withParam` could not be evaluated.
- A failed step group that is followed by an `Omitted` empty step group.

When a hook error stops a template, steps and tasks that had not started have no node, so `argo retry` runs them.
`argo retry` also creates the items of an expanded step or task that the hook error stopped.
See [Lifecycle-Hook](lifecyclehook.md) for what a hook error does.

A succeeded step or task can have an exit hook with its own `retryStrategy`.
After `argo retry`, the controller completes that exit hook without starting a new attempt of the hook.
`argo retry` does not run the exit hook of a succeeded step or task again when the hook has no `retryStrategy`.
