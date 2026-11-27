# Step Level Memoization

## Introduction

Workflows often have outputs that are expensive to compute.
Memoization reduces cost and workflow execution time by recording the result of previously run steps:
it stores the outputs of a template into a specified cache with a variable key.

You can memoize any step, whether or not it has outputs.

## Cache Backends

Argo Workflows supports two backends for storing memoization cache entries.
Both backends scope caches to the controller's namespace, so workflows that the controller manages can share a cache across workflow namespaces.

### ConfigMap (default)

The controller stores cached data in Kubernetes ConfigMaps by default.
You can manipulate cache entries through `kubectl` and the Kubernetes API.
The controller uses only ConfigMaps with the label `workflows.argoproj.io/configmap-type: Cache` as caches.
This prevents accidental access to other ConfigMaps in the system.

### SQL Database

> v4.2 and after

Alternatively, configure the controller to store cache entries in PostgreSQL or MySQL to avoid the 1 MiB ConfigMap size limit.
The workflow still specifies `memoize.cache.configMap.name`, but the SQL backend uses it as a logical cache name and creates no ConfigMap.

To enable SQL-backed memoization, add a `memoization` block to `workflow-controller-configmap`.
These examples use the databases and credential Secrets from the PostgreSQL and MySQL quick-start manifests.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
  namespace: argo
data:
  memoization: |
    tableName: cache_entries
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

For MySQL:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
  namespace: argo
data:
  memoization: |
    tableName: cache_entries
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

#### Initialization and configuration changes

The SQL backend stores entries in the table you specify with `memoization.tableName`, which defaults to `cache_entries`.
Table names must start with a lowercase letter, contain only lowercase letters, digits or underscores, and be at most 48 characters long.
The database connection settings remain under `postgresql` or `mysql`.

Only the elected leader connects to and migrates the memoization database, before processing workflows.
Standby replicas do not open memoization connection pools or run migrations.
The leader exits if it cannot initialize the configured database; it does not fall back to ConfigMaps.

When `memoization` changes, the controller initializes a replacement backend before publishing it.
For unrelated configuration edits, the controller reuses the existing backend without reconnecting or migrating.
It lets in-flight operations finish on the previous backend before closing the old connection pool.
When you remove `memoization`, the controller resumes ConfigMap-backed caching.
It does not copy existing entries between backends.

To manage the schema yourself, set `memoization.skipMigration: true` and apply the SQL in [Database Migrations](database-migrations.md#memoization-database).
The controller names the index `<tableName>_i1` and the migration history table `<tableName>_schema_history`.
If you later enable automatic migrations, the history table must contain the highest completed step in its `schema_version` integer primary-key column, as described on that page.

#### Sharing a database

The SQL cache key contains the controller namespace, cache name and cache key; it does not contain a cluster identifier.
Controllers in different clusters with the same namespace and table read and overwrite each other's entries.
Garbage collection applies to all expired entries in the table, across namespaces.
Use a separate database or a distinct `memoization.tableName` for each cluster when you need isolation.

#### Garbage collection

The SQL garbage collector prunes expired entries when the leader starts, then periodically.
Configure the interval in seconds with `MEMO_CACHE_GC_PERIOD_SECONDS`, which defaults to `86400` (24 hours).
Set it to `0` or a negative integer to disable both initial and periodic pruning; expired entries still produce cache misses.
The controller rejects malformed or overflowing values during construction, before leader election.

## Using Memoization

Configure memoization at the template level via `memoize`.
You must specify an explicit `key`, which can be a static string but usually depends on inputs.
You must also specify `cache.configMap.name` for either backend.
Optionally, set `maxAge` to limit the age of usable entries, as described in [Cache expiry](#cache-expiry).

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: memoized-workflow-
spec:
  entrypoint: print-message
  arguments:
    parameters:
      - name: message
        value: hello
  templates:
    - name: print-message
      inputs:
        parameters:
          - name: message
      memoize:
        key: "{{inputs.parameters.message}}"
        maxAge: "10s"
        cache:
          configMap:
            name: print-message-cache
      container:
        image: argoproj/argosay:v2
        command: [echo]
        args: ["{{inputs.parameters.message}}"]
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `key` | Yes | The explicit cache lookup key. |
| `cache` | Yes | The cache name; see [Cache Backends](#cache-backends). |
| `maxAge` | No | The maximum usable entry age; see [Cache expiry](#cache-expiry). |

### Cache expiry

`maxAge` is a duration such as `180s` or `24h`.
The controller treats entries older than the requesting template's `maxAge` as cache misses.
When the template starts, the controller saves the resolved value in node status.
At completion, it uses that value without re-evaluating expressions or requiring the original template to remain available.
This also applies to inline templates.

The SQL backend records expiry at save time using that resolved `maxAge`.
If a later request specifies a shorter `maxAge`, the controller can reject an entry before its stored expiry.
If you increase `maxAge`, the SQL backend does not extend an existing entry's stored expiry.
If you omit `maxAge`, the controller applies no age limit at lookup.
The SQL backend then records no expiry, so its garbage collector does not remove the entry.
There is no default TTL.

The controller checks ConfigMap cache freshness at lookup without storing an expiry timestamp.
Use `CACHE_GC_PERIOD` and `CACHE_GC_AFTER_NOT_HIT_DURATION` to control the existing unused-entry garbage collection for ConfigMap caches.

[Find a simple example for memoization here](https://github.com/argoproj/argo-workflows/blob/main/examples/memoize-simple.yaml).

!!! Note
    To use the ConfigMap backend, add `create` and `update` to the `configmaps` resource for the appropriate (cluster) roles.
    For a cluster install, update the `argo-cluster-role` cluster role.
    For a namespace install, update the `argo-role` role.
    The SQL backend does not require these ConfigMap permissions.

### Cache failures

The SQL cache retries reads and writes after transient connection, failover and transaction errors, using bounded exponential backoff.
Cancellation stops retries.
The controller reports permanent errors and transient errors that exhaust retries to the node.
It does not treat those errors as cache misses or switch to ConfigMaps.

If the controller cannot save a template's result to the cache, the node ends `Error`.
This applies to every memoized template, including DAG, Steps, HTTP and plugin templates, on both backends.
In v4.2, HTTP and plugin nodes also end `Error` on save failure; earlier versions only logged those failures.
For example, saving can fail if a ConfigMap exceeds 1 MiB or the controller lacks permission to update it.

When the controller serves a memoized template from the cache, it still emits the template's metrics (`metrics.prometheus`), with a duration close to zero.

## FAQ

1. If you see errors like `error creating cache entry: ConfigMap \"reuse-task\" is invalid: []: Too long: must have at most 1048576 characters`,
   this is due to [the 1 MiB ConfigMap size limit](https://github.com/kubernetes/kubernetes/issues/19781).
   These changes can help:
    * Delete the existing ConfigMap cache or use a different cache.
    * Reduce the size of the memoized output parameters.
    * Split the cache into different memoization keys and cache names.
    * Use the SQL backend to avoid the ConfigMap size limit.
