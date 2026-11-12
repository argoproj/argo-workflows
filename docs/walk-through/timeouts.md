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
    The controller enforces the workflow-level `activeDeadlineSeconds`, so it also ends a workflow whose pods have finished without their WorkflowTaskResult ever reaching the controller.
    A template-level `activeDeadlineSeconds` can't do that, because it becomes the pod's own `activeDeadlineSeconds`, and Kubernetes stops enforcing that once the pod has finished.
    Set the workflow-level field as well if you need the workflow itself to time out.
