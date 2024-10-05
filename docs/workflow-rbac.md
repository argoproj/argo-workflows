# Workflow RBAC

All pods in a workflow run with the service account specified in `workflow.spec.serviceAccountName`, or if omitted,
the `default` service account of the workflow's namespace. The amount of access which a workflow needs is dependent on
what the workflow needs to do. For example, if your workflow needs to deploy a resource, then the workflow's service
account will require 'create' privileges on that resource.

**Warning**: We do not recommend using the `default` service account in production. It is a shared account so may have
permissions added to it you do not want. Instead, create a service account only for your workflow.

The minimum for the executor to function:

For >= v3.4:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: executor
rules:
  - apiGroups:
      - argoproj.io
    resources:
      - workflowtaskresults
    verbs:
      - create
      - patch
```
