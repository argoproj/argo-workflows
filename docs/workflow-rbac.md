# Workflow RBAC

All pods in a workflow run with the service account specified in `workflow.spec.serviceAccountName`, or if omitted,
the `default` service account of the workflow's namespace. The amount of access which a workflow needs is dependent on
what the workflow needs to do. For example, if your workflow needs to deploy a resource, then the workflow's service
account will require 'create' privileges on that resource.

The bare minimum for a workflow running using the Emissary executor to function is outlined below:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: workflow-role
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - patch
```

If you are using another executor, or using resource template, you'll need additional permissions,
see [workflow-role](https://github.com/argoproj/argo-workflows/blob/master/manifests/quick-start/base/workflow-role.yaml)
.
