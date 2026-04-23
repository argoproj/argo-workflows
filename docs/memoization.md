# Step Level Memoization

## Introduction

Workflows often have outputs that are expensive to compute.
Memoization reduces cost and workflow execution time by recording the result of previously run steps:
it stores the outputs of a template into a specified cache with a variable key.

All steps can be memoized, whether or not they have outputs.

## Cache Backends

Argo Workflows supports two backends for storing memoization cache entries:

### ConfigMap (default)

By default, cached data is stored in Kubernetes ConfigMaps.
This allows you to easily manipulate cache entries manually through `kubectl` and the Kubernetes API without having to go through Argo.
All cache ConfigMaps must have the label `workflows.argoproj.io/configmap-type: Cache` to be used as a cache. This prevents accidental access to other important ConfigMaps in the system.

### SQL Database

> v4.0 and after

Alternatively, cache entries can be stored in a PostgreSQL or MySQL database. This is recommended for production use — it has no size limits, supports long-term persistence, and includes automatic garbage collection.

To enable SQL-backed memoization, add a `memoization` block to the `workflow-controller-configmap`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
  namespace: argo
data:
  memoization: |
    tableName: memoization_cache
    postgresql:
      host: postgres
      port: 5432
      database: postgres
      userNameSecret:
        name: argo-postgres-config
        key: username
      passwordSecret:
        name: argo-postgres-config
        key: password
```

Each cache entry computes an `expires_at` timestamp at save time from the template's `maxAge` field. If `maxAge` is not specified on the template, it defaults to 30 days (2592000 seconds). This default can be overridden by setting the `DEFAULT_MAX_AGE` environment variable on the workflow controller (accepts Go duration strings like `720h` or integer seconds like `2592000`).

The garbage collector periodically deletes entries whose `expires_at` has elapsed. The GC period defaults to 24 hours and can be configured via the `MEMO_CACHE_GC_PERIOD` environment variable.

MySQL is also supported:

```yaml
  memoization: |
    tableName: memoization_cache
    mysql:
      host: mysql
      port: 3306
      database: argo
      userNameSecret:
        name: argo-mysql-config
        key: username
      passwordSecret:
        name: argo-mysql-config
        key: password
```

## Using Memoization

Memoization is configured at the template level via the `memoize` field.

Memoization is set at the template level. You must specify a `key`, which can be static strings but more often depend on inputs.
You must also specify a name for the `config-map` cache.
Optionally you can set a `maxAge` in seconds or hours (e.g. `180s`, `24h`) to define how long should it be considered valid. If an entry is older than the `maxAge`, it will be ignored.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
   generateName: memoized-workflow-
spec:
   entrypoint: print-message
   templates:
      - name: print-message
        memoize:
           key: "{{inputs.parameters.message}}"
           maxAge: "10s"
           cache:
              configMap:
                 name: print-message-cache
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `key` | Yes | The cache lookup key. |
| `cache` | Yes | Specifies the cache storage. When using the ConfigMap backend, a ConfigMap is created. When using the SQL backend, `cache.configMap.name` acts as a logical group name in the database — no ConfigMap is created. |
| `maxAge` | Yes | Maximum age of a cache entry (e.g. `"180s"`, `"24h"`). Entries older than this are treated as misses at lookup time. |

[Find a simple example for memoization here](https://github.com/argoproj/argo-workflows/blob/main/examples/memoize-simple.yaml).

!!! Note
    To use memoization with the ConfigMap backend, add the verbs `create` and `update` to the `configmaps` resource for the appropriate (cluster) roles. For a cluster install, update the `argo-cluster-role` cluster role; for a namespace install, update the `argo-role` role. This is not required when using the SQL database backend.

If the controller cannot save a template's result to the cache, the template's node ends `Error`.
This applies to every memoized template, including DAG and Steps templates.
For example, this happens when the cache ConfigMap would be larger than the 1 MiB limit, or when the controller is not allowed to update the ConfigMap.

When the controller serves a memoized template from the cache, it still emits the template's metrics (`metrics.prometheus`), with a duration close to zero.

## FAQ

1. If you see errors like `error creating cache entry: ConfigMap \"reuse-task\" is invalid: []: Too long: must have at most 1048576 characters`,
   this is due to [the 1MB limit placed on the size of `ConfigMap`](https://github.com/kubernetes/kubernetes/issues/19781).
   Here are a couple of ways that might help resolve this:
    * Delete the existing `ConfigMap` cache or switch to use a different cache.
    * Reduce the size of the output parameters for the nodes that are being memoized.
    * Split your cache into different memoization keys and cache names so that each cache entry is small.
    * Switch to the SQL database backend to avoid the ConfigMap size limit.
