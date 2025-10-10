# Managing Synchronization Limits via API

This page explains how to manage synchronization limits (semaphores) using the Argo Server API and CLI.

## Overview

Argo Workflows provides two ways to configure synchronization limits:

1. **ConfigMap-based limits** (always available) - Define limits in Kubernetes ConfigMaps
2. **Database-based limits** (requires configuration) - Store limits in a shared database for cross-cluster synchronization
Use the API/CLI approach when you need to:

- Adjust semaphore limits dynamically without redeploying ConfigMaps
- Manage limits programmatically or from CI/CD pipelines
- Provision synchronization limits as part of automated infrastructure setup
For CLI usage, refer to the [CLI documentation](cli/argo_sync.md).
For API specifications, see the [Swagger documentation](swagger.md).

## ConfigMap-based Limits

ConfigMap-based limits are always available through the API and CLI.
No additional configuration is required.
The API allows administrators to create, read, update, and delete semaphore configurations stored in ConfigMaps without manually editing YAML files.
This is controlled via standard kubernetes RBAC.

## Database-based Limits

Database-based limits allow multiple workflow controllers (typically across different clusters) to share synchronization state.

### Prerequisites

Before you can manage database limits via the API, you must:

1. Configure a PostgreSQL or MySQL database for synchronization (see [workflow synchronization](synchronization.md#database-configuration))
2. Enable the synchronization API in your workflow controller configuration

### Enable the API

Add this configuration to your [workflow-controller-configmap.yaml](workflow-controller-configmap.yaml):

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  config: |
    synchronization:
      enableAPI: true
      # Database configuration is also required - see synchronization.md
```

!!! Warning
    Setting `enableAPI: true` only enables the API endpoints.
    You must also configure the database connection settings as described in the [synchronization documentation](synchronization.md#database-configuration).
!!! Warning
    Deleting a semaphore that is currently in use is allowed.
 Workflows attempting to take it after deletion will error.

## Permissions

### ConfigMap Limits

To manage ConfigMap-based limits, users need Kubernetes RBAC permissions to create, read, update, or delete ConfigMaps in the target namespace.
The API server enforces these permissions through standard Kubernetes RBAC.

### Database Limits

Database limits are not backed by Kubernetes resources, so Kubernetes RBAC cannot directly control access to them.
Instead, the Argo Server uses `workflow` permissions as a proxy:

- To **create** a database limit: requires permission to create workflows in the namespace
- To **get** a database limit: requires permission to get workflows in the namespace
- To **update** a database limit: requires permission to update workflows in the namespace
- To **delete** a database limit: requires permission to delete workflows in the namespace
For example, a user with this RBAC policy:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: workflow-operator
rules:
  - apiGroups: ["argoproj.io"]
    resources: ["workflows"]
    verbs: ["create", "get", "update", "delete"]
```

can perform all operations on database semaphores in that namespace.
