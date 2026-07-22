# Upgrading Guide

For the upgrading guide to a specific version of workflows change the documentation version in the lower right corner of your browser.

Breaking changes  typically (sometimes we don't realise they are breaking) have "!" in the commit message, as per
the [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/#summary).

## Upgrading to v4.1

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
