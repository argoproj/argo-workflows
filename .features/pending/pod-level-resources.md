Description: Pod-level resource requests and limits for workflow pods
Authors: [Isitha Subasinghe](https://github.com/isubasinghe)
Component: General
Issues: 16399

Workflow pods can now set [pod-level resource requests and limits](https://kubernetes.io/docs/tasks/configure-pod-container/assign-pod-level-resources/) via the new `podResources` field, available at the workflow spec level and the template level.
Template-level `podResources` overrides the workflow-level value.
This lets you set a single resource budget shared by all containers in a pod (main, init, wait and sidecars) instead of sizing each container individually.
Requires the `PodLevelResources` feature gate to be enabled on the cluster (beta and on by default since Kubernetes v1.34).
If the feature gate is disabled, the API server strips the field and the controller emits a `PodLevelResourcesDropped` warning event on the workflow.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: pod-level-resources-
spec:
  entrypoint: main
  podResources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: "1"
      memory: 512Mi
  templates:
    - name: main
      container:
        image: busybox
        command: [echo, hello]
```
