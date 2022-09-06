---
name: Regression report
about: Create a regression report. Not for support requests.
labels: ['bug', 'regression']
---
## Checklist

<!-- Do NOT open an issue until you have: --> 

* [ ] Double-checked my configuration.
* [ ] Tested using `:latest` images.
* [ ] Attached the smallest workflow that reproduces the issue.
* [ ] Attached logs from the workflow controller.
* [ ] Attached logs from the wait container.

## Summary

What happened/what you expected to happen?

What version are you running?

## Diagnostics

Paste the smallest workflow that reproduces the bug. We must be able to run the workflow.

```yaml

```

```
# Logs from the workflow controller:
kubectl logs -n argo deploy/workflow-controller | grep ${workflow} 
```

```
# Logs from in your workflow's wait container, something like:
kubectl logs -c wait -l workflows.argoproj.io/workflow=${workflow},workflow.argoproj.io/phase!=Succeeded
```

---
<!-- Issue Author: Don't delete this message to encourage other users to support your issue! -->
**Message from the maintainers**:

Impacted by this regression? Give it a 👍. We prioritise the issues with the most 👍.
