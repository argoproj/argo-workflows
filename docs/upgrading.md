# Upgrading Guide

For the upgrading guide to a specific version of workflows change the documentation version in the lower right corner of your browser.

Breaking changes  typically (sometimes we don't realise they are breaking) have "!" in the commit message, as per
the [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/#summary).

## Upgrading to v4.2

### ContainerSet siblings are no longer terminated when one container fails

Previously, as soon as any container in a `containerSet` exited with a non-zero exit code, the controller terminated the whole pod, killing any sibling containers that were still running.
This effectively made every `containerSet` fail fast.
It was an unintended side effect of a rule added for single-container templates and was never a documented feature ([#16000](https://github.com/argoproj/argo-workflows/pull/16000)).

The pod is now only terminated once all main containers have exited.
Siblings of a failed container run to completion, and containers that depend on a failed container still fail once that dependency has exited.
The node is still marked as failed once all containers have finished.

There is no configuration option to restore the previous fail-fast behavior.
If your `containerSet` relied on a failing container stopping its siblings early, add explicit `dependencies` so those containers wait for it, or have the containers detect and act on the failure themselves.

Additionally, a container whose dependency is killed before it can report an exit code now ends with exit code 64 and the message `died without reporting exit code`, and the node is marked `Error` rather than `Failed`.

### DAG and Steps templates run on one engine

DAG and Steps templates are now executed by one shared engine, so both template types follow the same rules for readiness, `continueOn`, retries, hooks and omission.
Most workflows behave as before.
The changes below are deliberate; most of them apply a rule that one template type already had to the other.

An expanded step (one with `withItems`, `withParam` or `withSequence`) now has a `TaskGroup` node that holds its items, as an expanded DAG task already had.
The step group's children are the steps, and the items hang under the step's `TaskGroup`.

#### Errors and `continueOn`

A step that fails before it runs now fails only itself.
This covers a step whose arguments, `when` clause, `withParam` or items cannot be resolved, whose pod is rejected, or whose `podSpecPatch` cannot be applied.
The step's node ends `Error` and the other steps in its group still run; then the step group ends `Error` and the workflow fails, with a message that names the step and the cause.
`continueOn.error` on such a step now lets the template carry on, as it already did for a DAG task.
Previously a Steps template failed the step group before the steps listed after it started, whatever the step's `continueOn` said.

An expanded task or step whose items include both `Error` and `Failed` items now always ends `Error`.
Previously it took the phase of the last failing item, so the result depended on the order of the items.
This affects everything that reads the task's phase: `depends: "task.Failed"` is false and `task.Errored` is true, `continueOn.failed` no longer covers the task and `continueOn.error` does, and `{{workflow.status}}` in an exit handler can be `Error` where it was `Failed`.
In a Steps template `continueOn` now applies to the expanded step as a whole, so an expanded step with `continueOn.error` carries on even when one item `Failed` alongside an item that ended `Error`.

A task that follows a nested DAG or Steps template now sees that template's own phase, not the phases of the nodes inside it.
If a task inside the nested template failed but `continueOn` let the nested template succeed, an omitted or `when`-skipped task after it no longer fails the outer DAG.

A memoized DAG or Steps template whose result cannot be saved to the memoization cache now ends `Error`, as a memoized container template already did.
This happens, for example, when the cache ConfigMap would exceed the 1 MiB limit, or the controller is not allowed to update it.
Previously the error was only logged and the template succeeded.

#### Hooks

A task's or step's exit or lifecycle hook that ends `Error` now ends the DAG or Steps template it belongs to with `Error`.
This covers a hook that could not be started (for example its pod was denied by an admission webhook, or its expression could not be evaluated), a hook that timed out, and a hook that errored while it ran, such as one whose pod was deleted.
Once a hook has errored, no new task or step of that template starts; tasks that are already running finish, and then the template ends `Error` with the hook's message.
If another task has already failed, or a step's lifecycle hook errors while the step is still running (the step itself is then marked `Error`), the template ends `Failed` with that failure's message instead.
A hook that runs and ends `Failed` is still ignored, as before.
Previously a DAG ignored an exit hook that errored and went on to run the task's dependants.
`continueOn.error` on a task does not cover an error from its hooks: previously a DAG could still succeed when a lifecycle hook of a task with `continueOn.error` errored.

A hook whose pod is still `Pending` when the hook template's `timeout` or `pendingTimeout` passes is now recorded as `Error` rather than `Failed`, so it counts as a hook error in both template types.
Previously a DAG ignored a timed-out exit hook.
A timed-out workflow-level hook (`spec.hooks`) or workflow `onExit` node is also shown as `Error` now; the workflow's own outcome does not change.

An error in the expression of a workflow-level hook is now recorded on an `Error` hook node named `<workflow name>.hooks.<hook name>`.
When the workflow's entry node has already finished, it keeps its phase and the workflow's phase does not change; previously the error was written onto the entry node, which was changed to `Error`.

Lifecycle hooks on an expanded DAG task now run once per item, as they already did for an expanded step, so a hook's arguments can use `{{item}}`.
Previously a DAG ran them once for the whole `TaskGroup`.

A DAG task that did not run, because its `when` clause was false or its dependencies omitted it, no longer runs its lifecycle or exit hooks, as was already the case for steps.

A DAG task's lifecycle hooks now always run before its exit hook, as they already did for other DAG tasks and for steps.
Previously a task that finished in the same reconciliation that started it (for example a memoization cache hit, or a retried task whose attempt had already succeeded) ran its exit hook alongside its lifecycle hooks.
Its exit hook, and so its dependants, now wait for the lifecycle hooks; the final result is the same.

#### Expanded tasks and steps

`{{workflow.failures}}` no longer lists `TaskGroup` nodes, for DAG or Steps templates; it lists the failed items themselves.
Previously a failed expanded DAG task added an extra entry for its `TaskGroup`, with no message.

After the upgrade, the `TaskGroup` node of an expanded step shows no estimated duration until the next successful run from the same WorkflowTemplate or CronWorkflow under the new controller.
The workflow's own estimate is not affected.

A `withSequence` with a negative `count` now expands to no items.
Previously it produced a descending sequence from `start`.

When an expanded DAG task's `templateRef` refers to other tasks' outputs, its `TaskGroup` node (or its `Skipped` node when the expansion is empty) now records the resolved `templateRef`, as Steps templates already did.

#### Step groups

A step group is still created when it starts.
A group that follows a failed, stopped or timed-out group is now shown, as `Omitted`, together with its omitted steps; previously it did not appear.
After `failFast` stops a template, later groups are not created.

An empty step group (`- []`) that follows a group that did not succeed now ends `Omitted`, so `argo retry` can re-run the failed group.

A step group with a step that ended `Error`, whether before the step ran or while it ran, is now `Error` with the message `step group deemed errored due to child <step> error: <reason>`.
Previously a step that errored while it ran, for example because its pod was deleted, made its group `Failed` with the message `child '<node ID>' failed`.
The Steps template and the workflow still end `Failed`, with the group's message.

A daemon step that dies after its step group has finished now leaves that group `Succeeded`.
The daemon's node fails, the Steps template and the workflow fail with `child '<node ID>' failed`, and no further group starts.
Previously the step group was changed to `Failed` as well.
In a DAG, expanded daemon items that die after their `TaskGroup` has finished now fail the DAG, as a daemon task without items already did.

#### Outputs, metrics and messages

`globalName` outputs are now exported when the node that produces them finishes, so the workflow's global outputs hold the value from the node that finished last.
Memoized steps served from the cache, HTTP and plugin steps, and suspend steps resumed with `argo resume` now export them too (in a DAG, as soon as they finish rather than when the DAG finishes).
Previously a DAG exported its tasks' outputs again when it finished, in the order the tasks were declared, so an older value could overwrite a newer one.

An entrypoint DAG template's own `globalName` outputs are now exported to `workflow.status.outputs`, as a Steps template's already were ([#14767](https://github.com/argoproj/argo-workflows/issues/14767)).
If a task in the DAG sets a global with the same name, the DAG's own output now overwrites it when the DAG finishes.

The outputs of a DAG template can now refer to `workflow.outputs`, for example `valueFrom.expression: workflow.outputs.parameters.g` or `from: "{{workflow.outputs.artifacts.a}}"`, as the outputs of a Steps template already could.

Template metrics (`metrics.prometheus`) are now also emitted when a memoized template is served from the cache, with a duration close to zero.
Previously cache hits emitted no metrics.

A failed DAG template's node now has the message `child '<node ID>' failed`, naming its first failed task, as a Steps template's node already did.
A `retryStrategy.expression` on a DAG template that tests `lastRetry.message` now sees this message.

When a `retryStrategy.expression` fails to evaluate, the retried node still ends `Error` as before, with the expression's error as its message; its last attempt keeps its own phase (for example `Failed`) instead of also becoming `Error`.

A DAG task node whose task fails before it runs (for example because its pod is rejected) now shows the cause without the `task '<node name>' errored:` prefix, as step nodes already did.

When the entrypoint DAG or Steps template ends `Error` because of its own error (for example its outputs cannot be resolved), the workflow message is now that error, without the `error in entry template execution:` prefix.

When a `containerSet` pod is deleted, containers that had already finished now keep their phase instead of becoming `Error` with the message `container deleted`; the pod's node is `Error`, and `argo retry` re-runs the pod.

#### Retry and rollback

`argo retry` now re-runs a failed DAG task whose dependants were all omitted, for example a task whose `when` clause or `withParam` could not be evaluated.
Previously the retry did not reset such a task.

If you roll the controller back to an earlier version while Steps workflows with an expanded step are running, those workflows stay `Running`, and `argo retry` does not recover them.
Let them finish before rolling back, or delete (or terminate) them and resubmit them afterwards.

## Upgrading to v4.1.2

### SSO users are logged out once on upgrade

The SSO session token format has changed to fix the oversized `authorization` cookie that broke SSO login in v4.1.0 and v4.1.1 ([#16748](https://github.com/argoproj/argo-workflows/pull/16748)).
Existing session cookies fail to validate after the upgrade, so users are redirected to log in again once; new sessions work as normal.
CLI users who saved a token from `argo auth token` need to re-fetch it.
No configuration or secret changes are required.

## Upgrading to v4.1

### Controller caches no longer store `managedFields`

The workflow-controller (and the parts of the argo-server that share its informers) now strips `metadata.managedFields` from objects before storing them in its informer caches ([#16563](https://github.com/argoproj/argo-workflows/pull/16563)).
This reduces controller memory usage — As a rough guide `managedFields` might be about 20% of the memory usage of objects, and at scale informer objects use most of the controllers memory footprint.
This is internal to the controller's caches; objects stored in the cluster are unaffected.

### `argo archive` commands accept a workflow name as well as a UID

The `argo archive get`, `delete`, `resubmit` and `retry` commands previously took a workflow UID as their argument.
They now accept either a workflow name or a UID ([#15198](https://github.com/argoproj/argo-workflows/pull/15198)).
An argument that matches the UUID format is treated as a UID; anything else is treated as a name, resolved within the selected namespace.
If multiple archived workflows share the name, the command fails and lists the matching UIDs.
You can force the interpretation with the new `--uid` or `--name` flags, for example if a workflow's name is itself formatted as a UUID.
Existing scripts that pass UIDs continue to work unchanged.

### INFORMER_WRITE_BACK environment variable removed

The `INFORMER_WRITE_BACK` environment variable has been removed.
This variable controlled whether to write workflow updates back to the informer cache (`true`) or sleep for 1 second (`false`, the default) after persisting updates.
Alternative mechanisms now prevent reprocessing, making both behaviors unnecessary.
If you have this variable set, it can be safely removed from your configuration.

## Upgrading to v4.0.7 and v3.7.16

### Outputs of skipped and omitted steps and tasks now resolve

In v4.0.6, v3.7.15, and earlier, referencing an output parameter of a step or task that was Skipped (its `when` condition was false) or Omitted (its `depends` condition was not satisfied) could leave the consumer stuck: simple tag references requeued forever and expression references failed.

These references now resolve deterministically.
If the producing template declares a `valueFrom.default` for the output, references resolve to that default.
Otherwise the output is treated as absent: an argument that is purely such a reference lets the consuming input's `default` apply, and expression tags see `nil` so `??` fallbacks work.
A reference that handles the absence in none of these ways — a simple tag such as `{{tasks.producer.outputs.parameters.msg}}` with no consumer input default, or an expression that does not handle the `nil` (for example a bare `{{= tasks.producer.outputs.parameters.msg}}` without `??`) — fails the node with a terminal error instead of leaving the workflow stuck.
To handle the absence, declare a `valueFrom.default` on the producer's output, a `default` on the consuming input, or use a `??` expression fallback.
This applies uniformly wherever such a reference appears, including `spec.volumes` and artifact `subPath` fields; only steps and tasks whose own `when` evaluates to false tolerate unhandled absent references, since they never run.

See [Outputs of Skipped and Omitted Nodes](variables.md#outputs-of-skipped-and-omitted-nodes) for the full rules.

There is no configuration flag or environment variable to opt out: the new behavior applies to every workflow on upgrade.
To keep the old empty-string behavior, handle the absence as described above (a producer `valueFrom.default`, a consumer input `default`, or a `??` fallback) before upgrading.

### Template output parameter expressions that evaluate to `nil` now fail

This change also applies when no node was skipped or omitted.
A template `outputs.parameters` entry whose `valueFrom.expression` evaluates to `nil` previously produced the literal string `<nil>`.
It now fails the node with a terminal error unless that output parameter declares a `valueFrom.default`.
Expressions can return `nil` even when the referenced steps all ran, for example a missing map key (`someMap['absent']`) or a `find()` with no match.
To keep a value, declare a `valueFrom.default` on the output parameter, or rewrite the expression to handle `nil` (for example with `??`).

## Upgrading to v4.0

### Deprecations

Several features were marked for deprecation in 3.6, and are now removed:

* The Python SDK is removed, we recommend migrating to [Hera](https://github.com/argoproj-labs/hera)
* `schedule` in CronWorkflows, `podPriority`, `mutex` and `semaphore` in Workflows and WorkflowTemplates.

For more information on how to migrate these see [deprecations](deprecations.md)

### Python SDK Removed

The Python SDK (`argo-workflows` package on PyPI) has been removed from the repository in version 4.0 as previously announced in v3.6.

If you have the Python SDK installed, it will mostly continue to work with Argo Workflows 4.0, but it will not receive updates, bug fixes, or support.
We recommend migrating to [Hera](https://github.com/argoproj-labs/hera), which is the recommended Python SDK for Argo Workflows.
Hera provides a more intuitive and Pythonic interface for working with Argo Workflows.

For migration guidance and documentation, see:

* [Hera Documentation](https://hera.readthedocs.io/)
* [Hera Quick Start Guide](https://hera.readthedocs.io/en/stable/walk-through/quick-start/)

### Logging levels

The logging levels available have been reduced to `debug`, `info`, `warn` and `error`.
Other levels will be mapped to their equivalent if you use them, although they were previously undocumented.

### Full CRDs

The [official release manifests](installation.md#official-release-manifests) now default to using CRDs with full validation information.
This enables using [Validating Admission Policy](https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/) and `kubectl explain ...` on Argo CRDs.

Existing installations using the [minimal CRDs](https://github.com/argoproj/argo-workflows/tree/main/manifests/base/crds/minimal) will continue to work, but you'll be unable to use features that rely on CRD validation information.

Use the following command to selectively apply the full CRDs for an existing installation:

```bash
kubectl apply --server-side --kustomize https://github.com/argoproj/argo-workflows/manifests/base/crds/full?ref=v4.0.0
```

### Go language (Developers)

If you are importing argo-workflows code into your go project you need to be aware of some changes.

Many go-lang functions have changed signature to require a [context](https://pkg.go.dev/context) as the first parameter.
In almost all cases you will need to provide a logger in your context.
The details are in [logging.go](https://github.com/argoproj/argo-workflows/blob/main/util/logging/logging.go)

The kubernetes client does not require a context.
The API client from [apiclient](https://github.com/argoproj/argo-workflows/blob/main/pkg/apiclient/apiclient.go) is an exception and will create a logger for you if you don't provide one.

In particular:

* Your logger must conform to the logging interface `Logger` from that file.
* Your logger should be retrievable from the context key `logger` (util/logging/logging.go `LoggerKey`)
* You may wish to use the logger from [slog.go](https://github.com/argoproj/argo-workflows/blob/main/util/logging/slog.go)

Apiclient no longer provides `NewClient` or `NewClientFromOpts`, you must use `NewClientFromOptsWithContext`.
