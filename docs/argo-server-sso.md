# Argo Server SSO

![alpha](assets/alpha.svg)

> v2.9 and after

## To start Argo Server with SSO.

Firstly, configure both SSO  settings [workflow-controller-configmap.yaml](workflow-controller-configmap.yaml) with the correct OAuth 2 values.

You must annotated at least one service account with:

* `workflows.argoproj.io/rbac-order: 1`: indicate the priority order.
* `workflows.argoproj.io/rbac-groups: groups`: which groups this applies to

Optionally, one service account with:

* `workflows.argoproj.io/rbac-default: "true"` to indicate it is the default account.

Example:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    workflows.argoproj.io/rbac-order: "1"
  name: argo-server
```

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    workflows.argoproj.io/rbac-default: "true"
  name: argo-server-read-only
```

Then, start the Argo Server using the SSO [auth mode](argo-server-auth-mode.md):

```
argo server --auth-mode sso --auth-mode ...
```
