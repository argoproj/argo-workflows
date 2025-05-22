# Workflow Actor

> v3.6.5 and after

If argo workflow has setup [SSO](argo-server-sso.md), when you perform an action on the argo workflow related resource (Workflow, CronWorkflow, WorkflowTemplate, ClusterWorkflowTemplate) via the CLI or UI, an attempt will be made to label it with the user and their action.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: my-wf
  labels:
    workflows.argoproj.io/actor: admin
    # labels must be DNS formatted, so the "@" is replaces by '.at.'  
    workflows.argoproj.io/creator-email: admin.at.your.org
    workflows.argoproj.io/actor-preferred-username: admin-preferred-username
    workflows.argoproj.io/action: Update
```

Available actions:

- Update
- Suspend
- Stop
- Terminate
- Resume

!!! NOTE
    Labels only contain `[-_.0-9a-zA-Z]`, so any other characters will be turned into `-`.
