---
name: Reproducible bug report 
about: Create a reproducible bug report. Not for support requests.
labels: ['bug', 'triage']
---

<!-- If this is a regression, please open a regression report instead. -->

## Checklist

<!-- Do NOT open an issue until you have: --> 

* [ ] Double-checked my configuration.
* [ ] Tested using the latest version.
* [ ] Used the Emissary executor.

## Summary

What happened/what you expected to happen?

What version are you running?

## Diagnostics

Paste the smallest workflow that reproduces the bug. We must be able to run the workflow.

```yaml

```

```bash
# Logs from the workflow controller:
kubectl logs -n argo deploy/workflow-controller | grep ${workflow} 

# If the workflow's pods have not been created, you can skip the rest of the diagnostics.

# The workflow's pods that are problematic:
kubectl get pod -o yaml -l workflows.argoproj.io/workflow=${workflow},workflow.argoproj.io/phase!=Succeeded

# Logs from in your workflow's wait container, something like:
kubectl logs -c wait -l workflows.argoproj.io/workflow=${workflow},workflow.argoproj.io/phase!=Succeeded
```

---
<!-- Issue Author: Don't delete this message to encourage other users to support your issue! -->
**Message from the maintainers**:

Impacted by this bug? Give it a 👍. We prioritise the issues with the most 👍.
