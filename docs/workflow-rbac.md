# Workflow RBAC

All pods in a workflow run with the service account specified in `workflow.spec.serviceAccountName`,
or if omitted, the `default` service account of the workflow's namespace. The amount of access which
a workflow needs is dependent on what the workflow needs to do. For example, if your workflow needs
to deploy a resource, then the workflow's service account will require 'create' privileges on that
resource.

The bare minimum for a workflow to function is outlined below:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: workflow-role
rules:
# pod get/watch is used to identify the container IDs of the current pod
# pod patch is used to annotate the step's outputs back to controller (e.g. artifact location)
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
  - watch
  - patch
# logs get/watch are used to get the pods logs for script outputs, and for log archival
- apiGroups:
  - ""
  resources:
  - pods/log
  verbs:
  - get
  - watch
```
