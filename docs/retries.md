# Retries

Argo Workflows offers a range of options for retrying failed steps.

## Retry policies

Use `retryPolicy` to choose which failures to retry:

- Always: Retry all failed steps
- OnFailure: Retry steps whose main container is marked as failed in Kubernetes
- OnError: Retry steps that encounter Argo controller errors, or whose init or wait containers fail
- OnTransientError: Retry steps that encounter errors [defined as transient](https://github.com/argoproj/argo-workflows/blob/master/util/errors/errors.go), or errors matching the TRANSIENT_ERROR_PATTERN [environment variable](https://argoproj.github.io/argo-workflows/environment-variables/).

## Retry expressions

You can also use `expression` to control retries. The `expression` field
accepts an [expr](https://github.com/antonmedv/expr) expression and has
access to the following variables:

- lastRetry.exitCode: The exit code of the last retry, of "-1" if not available
- lastRetry.status: The phase of the last retry: Error, Failed
- lastRetry.duration: The duration of the last retry, in seconds

If `expression` evaluates to false, the step will not be retried.

See [example](https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/retry-conditional.yaml) for usage.

## Backoff

You can configure the delay between retries with `backoff`. See [example](https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/retry-backoff.yaml) for usage.
