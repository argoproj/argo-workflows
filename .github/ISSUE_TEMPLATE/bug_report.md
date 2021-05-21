---
name: Reproducible bug report 
about: Create a reproducible bug report. Not for support requests.
labels: 'bug'
---
## Summary 

What happened/what you expected to happen?

## Diagnostics

What Kubernetes provider are you using? 

What version of Argo Workflows are you running? 

What executor are you running? Docker/K8SAPI/Kubelet/PNS/Emissary

Did this work in a previous version? I.e. is it a regression?

```yaml
Paste a workflow that reproduces the bug, including status:
kubectl get wf -o yaml ${workflow} 
```

```
Paste the logs from the workflow controller:
kubectl logs -n argo deploy/workflow-controller | grep ${workflow}
```

```
Paste the logs from your workflow's wait container:
kubectl logs -c wait -l workflows.argoproj.io/workflow=${workflow}
```

---
<!-- Issue Author: Don't delete this message to encourage other users to support your issue! -->
**Message from the maintainers**:

Impacted by this bug? Give it a 👍. We prioritise the issues with the most 👍.
