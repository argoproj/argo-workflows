# Security

## Argo Server Security

Argo Server implements security in three layers.

Firstly, you should enable [transport layer security](tls.md) to ensure your data cannot be read in transit.

Secondly, you should enable an [authentication mode](argo-server.md#auth-mode) to ensure that you do not run workflows from unknown users.

Finally, you should configure the `argo-server` role and role binding with the correct permissions.

### Read-Only

You can achieve this by configuring the `argo-server` role ([example](https://github.com/argoproj/argo/blob/master/manifests/namespace-install/argo-server-rbac/argo-server-role.yaml) with only read access (i.e. only `get`/`list`/`watch` verbs).

## Workflow Pod Security

See [workflow pod security context](workflow-pod-security-context.md).