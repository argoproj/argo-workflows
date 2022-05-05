# Suspending

Workflows can be suspended by

```bash
argo suspend WORKFLOW
```

Or by specifying a `suspend` step on the workflow:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: suspend-template-
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    steps:
    - - name: build
        template: whalesay
    - - name: approve
        template: approve
    - - name: delay
        template: delay
    - - name: release
        template: whalesay

  - name: approve
    suspend: {}

  - name: delay
    suspend:
      duration: "20"    # Must be a string. Default unit is seconds. Could also be a Duration, e.g.: "2m", "6h", "1d"

  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
```

Once suspended, a Workflow will not schedule any new steps until it is resumed. It can be resumed manually by

```bash
argo resume WORKFLOW
```

Or automatically with a `duration` limit as the example above.
