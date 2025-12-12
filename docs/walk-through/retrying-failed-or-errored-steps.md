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
* `nodeAntiAffinity` prevents running steps on the same host.  Current implementation allows only empty `nodeAntiAffinity` (i.e. `nodeAntiAffinity: {}`) and by default it uses label `kubernetes.io/hostname` as the selector.

Providing an empty `retryStrategy` (i.e. `retryStrategy: {}`) will cause a container to retry until completion.
