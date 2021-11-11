---
name: Reproducible bug report 
about: Create a reproducible bug report. Not for support requests.
labels: ['bug', 'triage']
---
<!--
Before we start, around 2/3 of issues can be fixed by one of the following:

* Have you double-checked your configuration? Maybe 30% of issues are wrong configuration.
* Have you tested to see if it is fixed in the latest version? Maybe 20% of issues are fixed by this.
* Have you tried using the PNS executor instead of Docker? Maybe 50% of artifact related issues are fixed by this.

If this is a regression, please open a regression report instead.
-->

## Summary

What happened/what you expected to happen?

What version of Argo Workflows are you running?

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

Impacted by this bug? Give it a 👍. We prioritise the issues with the most 👍.
