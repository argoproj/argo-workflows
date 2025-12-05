# Retrying Failed or Errored Steps

You can specify a `retryStrategy` that will dictate how failed or errored steps are retried:

/// tab | YAML

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
      image: python:alpine3.6
      command: ["python", -c]
      # fail with a 66% probability
      args: ["import random; import sys; exit_code = random.choice([0, 1, 1]); sys.exit(exit_code)"]
```

///

/// tab | Python

```python
from hera.workflows import Container, Workflow
from hera.workflows.models import (
    Backoff,
    RetryAffinity,
    RetryNodeAntiAffinity,
    RetryStrategy,
)

with Workflow(
    generate_name="retry-backoff-",
    entrypoint="retry-backoff",
) as w:
    Container(
        name="retry-backoff",
        retry_strategy=RetryStrategy(
            affinity=RetryAffinity(node_anti_affinity=RetryNodeAntiAffinity()),
            backoff=Backoff(
                duration="1", factor=2, max_duration="1m"
            ),
            limit=10,
            retry_policy="Always",
        ),
        image="python:alpine3.6",
        command=["python", "-c"],
        args=[
            "import random; import sys; exit_code = random.choice([0, 1, 1]); sys.exit(exit_code)"
        ],
    )
```

///

* `limit` is the maximum number of times the container will be retried.
* `retryPolicy` specifies if a container will be retried on failure, error, both, or only transient errors (e.g. i/o or TLS handshake timeout). "Always" retries on both errors and failures. Also available: `OnFailure` (default), "`OnError`", and "`OnTransientError`" (available after v3.0.0-rc2).
* `backoff` is an exponential back-off
* `nodeAntiAffinity` prevents running steps on the same host.  Current implementation allows only empty `nodeAntiAffinity` (i.e. `nodeAntiAffinity: {}`) and by default it uses label `kubernetes.io/hostname` as the selector.

Providing an empty `retryStrategy` (i.e. `retryStrategy: {}`) will cause a container to retry until completion.
