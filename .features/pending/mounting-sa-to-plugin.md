Description: Add support for mounting default ServiceAccount in executor plugin as a configurable option
Author: [https://github.com/ntny)
Component: General
Issues: 14744
This PR allows using the existing workflow ServiceAccount without additional requirements or the need to maintain both ServiceAccounts when they are supposed to have identical permissions.

This feature introduces:
- optional field `AutomountWorkflowServiceAccountToken` field in the executor plugin ConfigMap description

When `AutomountWorkflowServiceAccountToken` is set default SA from workflow is used.  


```yaml
data:
  sidecar.automountWorkflowServiceAccountToken: "true" # New parameter to enable mounting the workflow's default ServiceAccount token
  sidecar.container: |
    image: ....
    name: workflow-sa-plugin
```

```yaml
metadata:
  labels:
    workflows.argoproj.io/configmap-type: ExecutorPlugin
  name: demo-plugin
data:
  sidecar.automountServiceAccountToken: "true" # mounts the token for the 'demo-plugin-executor-plugin' ServiceAccount as before
  sidecar.container: |
    image: ...
    name: demo-plugin
```

```yaml
data:
  sidecar.automountServiceAccountToken: "true" 
  sidecar.automountWorkflowServiceAccountToken: "true"
  # ❌ Enabling both automountServiceAccountToken and automountWorkflowServiceAccountToken will result in an error with an appropriate message.
  sidecar.container: |
    image: ....
    name: workflow-sa-plugin
```
