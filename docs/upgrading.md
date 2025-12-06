# Upgrading Guide

For the upgrading guide to a specific version of workflows change the documentation version in the lower right corner of your browser.

Breaking changes  typically (sometimes we don't realise they are breaking) have "!" in the commit message, as per
the [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/#summary).

## Upgrading to v4.0

### Deprecations

Several features were marked for deprecation in 3.6, and are now removed:

* The Python SDK is removed, we recommend migrating to [Hera](https://github.com/argoproj-labs/hera)
* `schedule` in CronWorkflows, `podPriority`, `mutex` and `semaphore` in Workflows and WorkflowTemplates.

For more information on how to migrate these see [deprecations](deprecations.md)

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
