# Retries

Argo Workflows offers a range of options for retrying failed steps.

!!! Note "restarts"
    For infrastructure-level failures that occur before your container starts (like node evictions or disk pressure), see [Automatic Pod Restarts](pod-restarts.md).
    This page covers application-level retries using `retryStrategy`.

## Configuring `retryStrategy` in `WorkflowSpec`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: retry-container-
spec:
  entrypoint: retry-container
  templates:
  - name: retry-container
    retryStrategy:
      limit: "10"
    container:
      image: python:alpine3.23
      command: ["python", -c]
      # fail with a 66% probability
      args: ["import random; import sys; exit_code = random.choice([0, 1, 1]); sys.exit(exit_code)"]
```

The `retryPolicy` and `expression` are re-evaluated after each attempt. For example, if you set `retryPolicy: OnFailure` and your first attempt produces a failure then a retry will be attempted. If the second attempt produces an error, then another attempt will not be made.

## Retry policies

Use `retryPolicy` to choose which failure types to retry:

- `Always`: Retry all failed steps
- `OnFailure`: Retry steps whose main container is marked as failed in Kubernetes
- `OnError`: Retry steps that encounter Argo controller errors, or whose init or wait containers fail
- `OnTransientError`: Retry steps that encounter errors [defined as transient](https://github.com/argoproj/argo-workflows/blob/main/util/errors/errors.go), or errors matching the `TRANSIENT_ERROR_PATTERN` [environment variable](environment-variables.md). Available in version 3.0 and later.

The `retryPolicy` applies even if you also specify an `expression`, but in version 3.5 or later the default policy means the expression makes the decision unless you explicitly specify a policy.

The default `retryPolicy` is `OnFailure`, except in version 3.5 or later when an expression is also supplied, when it is `Always`. This may be easier to understand in this diagram.

```mermaid
flowchart LR
  start([Will a retry be attempted])
  start --> policy
  policy(Policy Specified?)
  policy-->|No|expressionNoPolicy
  policy-->|Yes|policyGiven
  policyGiven(Expression Specified?)
  policyGiven-->|No|policyGivenApplies
  policyGiven-->|Yes|policyAndExpression
  policyGivenApplies(Supplied Policy)
  policyAndExpression(Supplied Policy AND Expression)
  expressionNoPolicy(Expression specified?)
  expressionNoPolicy-->|No|onfailureNoExpr
  expressionNoPolicy-->|Yes|version
  onfailureNoExpr[OnFailure]
  onfailure[OnFailure AND Expression]
  version(Workflows version)
  version-->|3.4 or earlier|onfailure
  always[Only Expression matters]
  version-->|3.5 or later|always
```

An example retry strategy:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: retry-on-error-
spec:
  entrypoint: error-container
  templates:
  - name: error-container
    retryStrategy:
      limit: "2"
      retryPolicy: "Always"
    container:
      image: python
      command: ["python", "-c"]
      # fail with a 80% probability
      args: ["import random; import sys; exit_code = random.choice(range(0, 5)); sys.exit(exit_code)"]
```

## Conditional retries

> v3.2 and after

You can also use `expression` to control retries.
This is an [expr expression](variables.md#expression) with access to the following variables:

- `lastRetry.exitCode`: The exit code of the last retry as a string, or "-1" if not available

    ```yaml
    expression: asInt(lastRetry.exitCode) > 1 # Retry if code is greater than 1
    ```

- `lastRetry.status`: The phase of the last retry: Error, Failed

    ```yaml
    expression: lastRetry.status != "Error" # Retry if not an error
    ```

- `lastRetry.duration`: The duration of the last retry, in seconds

    ```yaml
    expression: asInt(lastRetry.duration) < 60 # Retry unless duration >= 1 minute
    ```

- `lastRetry.message`: The message output from the last retry (available from version 3.5)

    ```yaml
    # Retry if message matches the regular expression
    expression: lastRetry.message matches 'imminent node shutdown|pod deleted'
    ```

If `expression` evaluates to false, the step will not be retried.

The `expression` result will be logical *and* with the `retryPolicy`. Both must be true to retry.

Boolean operators can be used to combine multiple conditions. See [example](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/retry-conditional.yaml) for usage.

## Back-Off

You can configure the delay between retries with `backoff`. See [example](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/retry-backoff.yaml) for usage.
