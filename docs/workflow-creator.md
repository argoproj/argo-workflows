# Workflow Creator

> v2.9 and after

If you create your workflow via the CLI or UI, an attempt will be made to label it with the user who created it

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: my-wf
  labels:
    workflows.argoproj.io/creator: admin
    # labels must be DNS formatted, so the "@" is replaces by '.at.'  
    workflows.argoproj.io/creator-email: admin.at.your.org
    workflows.argoproj.io/creator-preferred-username: admin-preferred-username
```

!!! NOTE
    Labels only contain `[-_.0-9a-zA-Z]`, so any other characters will be turned into `-`.
