---
name: Bug report
about: Create a report to help us improve
labels: 'bug'
---
## Summary 

What happened/what you expected to happen?

## Diagnostics

What version of Argo Workflows are you running?

Paste a screenshot of your workflow.

```yaml
Paste here either:
  A workflow anyone can run that reproduces your bug (tip: we can't run private images). 
Or:
  The whole workflow here, including status, e.g. kubectl get wf ${workflow}
```

```
Paste the logs from the workflow controller:
kubectl logs -n argo $(kubectl get pods -l app=workflow-controller -n argo -o name) | grep ${workflow}
```

---
<!-- Issue Author: Don't delete this message to encourage other users to support your issue! -->
**Message from the maintainers**:

Impacted by this bug? Give it a 👍. We prioritise the issues with the most 👍.
