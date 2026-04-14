Description: Add pod-free `action: wait` with `waitFor: delete` for resource templates
Authors: [Andre Kurait](https://github.com/AndreKurait)
Component: General
Issues: 15874

The new `action: wait` with `waitFor: delete` allows workflows to block until a Kubernetes resource is fully deleted, without creating a pod.
The controller checks resource existence directly via the K8s API on each reconciliation cycle.

This eliminates pod overhead, OOM risk, and image pull costs for deletion-watching patterns.

```yaml
- name: wait-for-job-deleted
  resource:
    action: wait
    waitFor: delete
    manifest: |
      apiVersion: batch/v1
      kind: Job
      metadata:
        name: my-job
```

If the resource doesn't exist when the step starts, it succeeds immediately.
Use `activeDeadlineSeconds` to set a timeout.

`successCondition`, `failureCondition`, and outputs cannot be used with the `wait` action.
