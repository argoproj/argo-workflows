---
name: Regression report
about: Create a regression report. Not for support requests.
labels: ['bug', 'regression', 'triage']
---
## Summary

What happened/what you expected to happen?

What version is it broken in?

What version was it working in?

## Diagnostics

Either a workflow that reproduces the bug, or paste you whole workflow YAML, including status, something like:

```yaml
kubectl get wf -o yaml ${workflow}
```

What Kubernetes provider are you using?

What executor are you running? Docker/K8SAPI/Kubelet/PNS/Emissary

```bash
# Logs from the workflow controller:
kubectl logs -n argo deploy/workflow-controller | grep ${workflow}

# The workflow's pods that are problematic:
kubectl get pod -o yaml -l workflows.argoproj.io/workflow=${workflow},workflow.argoproj.io/phase!=Succeeded

# Logs from in your workflow's wait container, something like:
kubectl logs -c wait -l workflows.argoproj.io/workflow=${workflow},workflow.argoproj.io/phase!=Succeeded
```

---
<!-- Issue Author: Don't delete this message to encourage other users to support your issue! -->
**Message from the maintainers**:

Impacted by this regression? Give it a 👍. We prioritise the issues with the most 👍.
