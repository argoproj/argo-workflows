# Retrying Failed or Errored Steps

You can specify a `retryStrategy` that will dictate how failed or errored steps are retried:

```yaml
# This example demonstrates the use of retry back offs
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: retry-backoff-
spec:
  entrypoint: retry-backoff
  templates:
  - name: retry-backoff
    retryStrategy:
      limit: 10
      retryPolicy: "Always"
      backoff:
        duration: "1"      # Must be a string. Default unit is seconds. Could also be a Duration, e.g.: "2m", "6h", "1d"
        factor: 2
        maxDuration: "1m"  # Must be a string. Default unit is seconds. Could also be a Duration, e.g.: "2m", "6h", "1d"
      affinity:
        nodeAntiAffinity: {}
    container:
      image: python:alpine3.23
      command: ["python", -c]
      # fail with a 66% probability
      args: ["import random; import sys; exit_code = random.choice([0, 1, 1]); sys.exit(exit_code)"]
```

* `limit` is the maximum number of times the container will be retried.
* `retryPolicy` specifies if a container will be retried on failure, error, both, or only transient errors (e.g. i/o or TLS handshake timeout). "Always" retries on both errors and failures. Also available: `OnFailure` (default), "`OnError`", and "`OnTransientError`" (available after v3.0.0-rc2).
* `backoff` is an exponential back-off
* `nodeAntiAffinity` prevents running steps on the same host.
  By default it uses label `kubernetes.io/hostname` as the selector.
  Its `type` controls how strictly earlier hosts are avoided (available after v4.2.0).
    * `Required` is the default, and is what an empty `nodeAntiAffinity: {}` gives you.
      It excludes previously tried hosts outright, so a retry stays unschedulable once every eligible host has been tried.
    * `Preferred` only de-prioritises them.
      A retry can then still be scheduled onto a previously tried host once every eligible host has been tried.
      This is a preference rather than a guarantee, so other scheduling constraints can still leave the retry pending.
      Use it when the retry limit is higher than the number of eligible nodes.

```yaml
retryStrategy:
  limit: 10
  affinity:
    nodeAntiAffinity:
      type: Preferred
```

Providing an empty `retryStrategy` (i.e. `retryStrategy: {}`) will cause a container to retry until completion.
