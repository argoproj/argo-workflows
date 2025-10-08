# Synchronization

It is possible to create a lock configuration using the Argo Server API (and CLI). A ConfigMap API is always available and the database one is disabled by default.
For the API reference, please check the [Swagger documentation](swagger.md), and for CLI usage, refer to the [CLI documentation](cli/argo_sync.md).

## Enable Database Lock Configuration Using API

To control database lock configuration using the API, you must enable the API option in the `synchronization` section of [your configuration](workflow-controller-configmap.yaml) and set `enableAPI` to `true`:

```yaml
  synchronization:
    enableAPI: true
```

!!! Note
    This only enables the API, to enable a database lock, you also need to enable database lock configuration. For this, please refer to the [synchronization documentation](synchronization.md).

## Permissions

To create a ConfigMap lock, users need appropriate Kubernetes permissions, as this functionality relies on Kubernetes RBAC.

Since there is no dedicated permission system for database locks and no Kubernetes object available to validate RBAC for them, the Argo Server relies on `workflow` permissions. If a user is allowed to create a `workflow`, they are also allowed to create a database lock. The same logic applies to all other operations such as `update`, `get`, and `delete`.
