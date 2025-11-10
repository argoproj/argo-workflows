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

### Go language (Developers)

If you are importing argo-workflows code into your go project you need to be aware of some changes.

Many go-lang functions have changed signature to require a [context](https://pkg.go.dev/context) as the first parameter.
In almost all cases you will need to provide a logger in your context.
The details are in [logging.go](https://github.com/argoproj/argo-workflows/blob/main/util/logging/logging.go)

In particular:

* Your logger must conform to the logging interface `Logger` from that file.
* Your logger should be retrievable from the context key `logger` (util/logging/logging.go `LoggerKey`)
* You may wish to use the logger from [slog.go](https://github.com/argoproj/argo-workflows/blob/main/util/logging/logging.go)
