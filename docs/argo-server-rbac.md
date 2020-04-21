# Argo Server RBAC

![alpha](assets/alpha.svg)

> v2.9 and after

## To start the server with RBAC:

Firstly, you may wish to enable [SSO](argo-server-sso.md). The groups provided by your OAuth 2 provider and will be used as roles.

Then, configure RBAC in the [config map](workflow-controller-configmap.yaml).

Tip: use the [Casbin editor](https://casbin.org/editor/) to test out your policy. The model can be found in [rbac/service.go](../server/rbac/service.go).

The `act` (action) and `obj` (object) values are based on the Swagger operationId (which is in turn based on the gRPC methods). 

E.g.

* `CreateClusterWorkflowTemplate` -> `create` `clusterworkflowtemplate`
* `DeleteWorkflow` -> `delete` `workflow`

```
Examples:

```
# allow admin to do anything
p, admin, *, *

# allow reod-only to only get/list/watch
p, read-only, *, get
p, read-only, *, list
p, read-only, *, watch

# allow historians to only get/list workflows
p, historian, workflows, get
p, historian, workflows, list
```