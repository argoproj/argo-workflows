# Workflow Pod Security Context

By default, all workflow pods run as root. The Docker executor even requires `privileged: true`.

For other [workflow executors](workflow-executors.md), you can run your workflow pods more securely by configuring the [security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) for your workflow pod.

This is likely to be necessary if you have a [pod security policy](https://kubernetes.io/docs/concepts/policy/pod-security-policy/). You probably can't use the Docker executor if you have a pod security policy.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: security-context-
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 8737 #; any non-root user
```

You can configure this globally using [workflow defaults](default-workflow-specs.md).

!!! Warning "It is easy to make a workflow need root unintentionally"
    You may find that user's workflows have been written to require root with seemingly innocuous code. E.g. `mkdir /my-dir` would require root.

!!! Note "You must use volumes for output artifacts"
    If you use `runAsNonRoot` - you cannot have output artifacts on base layer (e.g. `/tmp`). You must use a volume (e.g. [empty dir](empty-dir.md)).
