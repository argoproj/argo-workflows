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
      - workflowtaskresult
    verbs:
      - create
      - patch
```

For <= v3.3 use.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: executor
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
      - patch
```

Warning: For many organisations, it may not be acceptable to give a workflow the `pod patch` permission, see [#3961](https://github.com/argoproj/argo-workflows/issues/3961)

If you are not using the emissary, you'll need additional permissions.
See [executor](https://github.com/argoproj/argo-workflows/tree/master/manifests/quick-start/base/executor) for suitable
permissions.
