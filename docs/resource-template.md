# Resource Template

> v2.0 and after

See [Kubernetes Resources](walk-through/kubernetes-resources.md).

## Wait for Resource Deletion

The `wait` action blocks until a resource is fully deleted.
Unlike other resource actions, it runs entirely in the controller — no pod is created.

```yaml
- name: wait-for-cm-deleted
  resource:
    action: wait
    waitFor: delete
    manifest: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: my-cm
```

If the resource doesn't exist when the step starts, it succeeds immediately.
Use `activeDeadlineSeconds` on the template to set a timeout.
