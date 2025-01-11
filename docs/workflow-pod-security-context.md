# Workflow Pod Security Context

By default, all workflow pods run as root.

You can run your workflow pods more securely by configuring the [security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) for your workflow pod.

This is likely to be necessary if pod security standards ([PSS](https://kubernetes.io/docs/concepts/security/pod-security-standards)) are enforced by
[PSA](https://kubernetes.io/docs/concepts/security/pod-security-admission/) or other means, or if you have a
[pod security policy](https://kubernetes.io/docs/concepts/policy/pod-security-policy/) (deprecated).

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

!!! Note "You must use volumes for output artifacts (v3.3 or earlier)"
    If you use `runAsNonRoot` in versions v3.3 or earlier, you cannot have output artifacts on base layer (e.g. `/tmp`). You must use a volume (e.g. [empty dir](empty-dir.md)).
    In versions later than v3.3, the [Emissary executor](workflow-executors.md#emissary-emissary)
    allows artifacts on the base layer with `runAsNonRoot`.
