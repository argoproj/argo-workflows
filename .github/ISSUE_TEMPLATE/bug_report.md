---
name: Reproducible bug report 
about: Create a reproducible bug report. Not for support requests.
labels: 'bug'
---
## Summary 

What happened/what you expected to happen?

Before we start, around 2/3 of issues can be fixed by one of the following:

* Have you double-checked your configuration? Maybe 30% of issues are wrong configuration.
* Have you tested to see if it is fixed in the latest version? Maybe 20% of issues are fixed by this.
* Have you tried using the PNS executer instead of Docker? Maybe 50% of artifact related issues are fixed by this.

## Diagnostics

👀 Yes! We need all of your diagnostics, please make sure you add it all, otherwise we'll go around in circles asking you for it:

What Kubernetes provider are you using? 

What version of Argo Workflows are you running? 

What executor are you running? Docker/K8SAPI/Kubelet/PNS/Emissary

Did this work in a previous version? I.e. is it a regression?

Are you pasting thousands of log lines? That's too much information. 

```bash
# Either a workflow that reproduces the bug, or paste you whole workflow YAML, including status, something like:
kubectl get wf -o yaml ${workflow}

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
