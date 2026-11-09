# Timeouts

You can use the field `activeDeadlineSeconds` to limit the elapsed time for a workflow:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: timeouts-
spec:
  activeDeadlineSeconds: 10 # terminate workflow after 10 seconds
  entrypoint: sleep
  templates:
  - name: sleep
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo sleeping for 1m; sleep 60; echo done"]
```

You can limit the elapsed time for a specific template as well:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: timeouts-
spec:
  entrypoint: sleep
  templates:
  - name: sleep
    activeDeadlineSeconds: 10 # terminate container template after 10 seconds
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo sleeping for 1m; sleep 60; echo done"]
```

!!! Note
    The workflow-level `activeDeadlineSeconds` is enforced by the controller, so it also
    bounds a workflow whose pods have already finished without their task result ever
    reaching the controller. A template-level `activeDeadlineSeconds` cannot do that: it is
    enforced inside the pod by the executor's deadline monitor, which cannot act once the
    containers have exited. Set the workflow-level field as well when you need that bound.
