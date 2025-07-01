# Synchronization

> v2.10 and after

You can limit the parallel execution of Workflows or Templates:

- You can use mutexes to restrict Workflows or Templates to only having a single concurrent execution.
- You can use semaphores to restrict Workflows or Templates to a configured number of parallel executions.
- You can use parallelism to restrict concurrent tasks or steps within a single Workflow.

The term "locks" on this page means mutexes and semaphores.

## Lock types

Argo supports local locks, and multiple controller locks.

### Local locks

Local locks are local to the controller that is running them, and only affect Workflows that are running on that controller.

To configure the size of local semaphores you should use ConfigMap.
No configuration is required for local mutexes.

A Workflow that uses a Workflow-level local mutex would look like this:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-wf-level-
  namespace: foo
spec:
  synchronization:
    mutexes:
      - name: bar
```

You can create semaphore configurations in a ConfigMap that can be referred to from a Workflow or Template.

For example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
 name: my-config
data:
  workflow: "1"  # Only one Workflow can run at given time in particular namespace
  template: "2"  # Two instances of Template can run at a given time in particular namespace
```

And a Workflow that uses this Workflow-level local semaphore would look like this:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-wf-level-
  namespace: foo
spec:
  synchronization:
    semaphores:
      - configMapKeyRef:
          key: bar
          name: my-config
```

### Multiple controller locks

Multiple controllers can share locks using a database as an intermediary.
This would normally be used to share locks across multiple clusters, but can also be used to share locks across multiple controllers in the same cluster.

To configure multiple controller locks, you need to set up a database (either PostgreSQL or MySQL) and [configure it](#database-configuration) in the workflow-controller-configmap ConfigMap.
All controllers which want to share locks must share all of these tables.
If you do not configure the database you will get an error if you try to use database locks.

A Workflow that uses a Workflow-level database mutex would look like this:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-wf-level-
  namespace: foo
spec:
  synchronization:
    mutexes:
      - database:
          key: bar
```

And a Workflow that uses a Workflow-level database semaphore would look like this:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: synchronization-wf-level-
  namespace: foo
spec:
  synchronization:
    semaphores:
      - database:
          key: bar
```

The semaphore size is configured in the [the limit table](#limit-table), with a name and size.
The semaphore name contains the namespace, and takes the standard kubernetes format of`<namespace>/<name>`.
As an example for a semaphore named `test` in namespace `devel` the name would be `devel/test`.
The above example Workflow would need something like

```sql
INSERT INTO sync_limit (name, sizelimit) VALUES ('foo/bar', 3);
```

#### Cluster Time

The time on the clusters must be synchronized.
The time-stamps put into the database are used to determine if a controller is responsive, and if the times on the clusters differ this will not work correctly.
The Workflow creation time-stamp is used to order Workflows in the queue, and if the times differ this will also not work correctly.

## Workflow-level Synchronization

You can limit parallel execution of Workflows by using the same synchronization reference.

In this example the synchronization key `workflow` is configured as limit `"1"`, so only one Workflow instance will execute at a time even if multiple Workflows are created.

Using a semaphore configured by a ConfigMap:

```yaml title="examples/synchronization-wf-level.yaml"
--8<-- "examples/synchronization-wf-level.yaml:12"
```

Using a mutex is equivalent to a limit `"1"` semaphore:

```yaml title="examples/synchronization-mutex-wf-level.yaml"
--8<-- "examples/synchronization-mutex-wf-level.yaml:3"
```

Using a multi-controller semaphore configured in a database:

```yaml title="examples/synchronization-db-wf-level.yaml"
--8<-- "examples/synchronization-db-wf-level.yaml:4"
```

Using a multi-controller mutex:

```yaml title="examples/synchronization-db-mutex-wf-level.yaml"
--8<-- "examples/synchronization-db-mutex-wf-level.yaml:3"
```

## Template-level Synchronization

You can limit parallel execution of Templates by using the same synchronization reference.

In this example the synchronization key `template` is configured as limit `"2"`, so a maximum of two instances of the `acquire-lock` Template will execute at a time.
This applies even when multiple steps or tasks within a Workflow or different Workflows refer to the same Template.

Using a semaphore configured by a ConfigMap:

```yaml title="examples/synchronization-tmpl-level.yaml"
--8<-- "examples/synchronization-tmpl-level.yaml:11"
```

Using a mutex will limit to a single concurrent execution of the Template:

```yaml title="examples/synchronization-mutex-tmpl-level.yaml"
--8<-- "examples/synchronization-mutex-tmpl-level.yaml:3"
```

Using a multi-controller semaphore configured in a database:

```yaml title="examples/synchronization-db-tmpl-level.yaml"
--8<-- "examples/synchronization-db-tmpl-level.yaml:5"
```

Using a multi-controller mutex:

```yaml title="examples/synchronization-db-mutex-tmpl-level.yaml"
--8<-- "examples/synchronization-db-mutex-tmpl-level.yaml:3"
```

## Namespaces

Each distinct lock is defined by a namespace and a key.
For a mutex the key is the same as the lock `name`.
For a semaphore the key is given by the `key` field in the lock definition.
The namespace defaults to the namespace of the Workflow, but can be overridden by the `namespace` field in the lock definition.

The namespace for local ConfigMap locks is also used to read the size of the semaphore from a ConfigMap in that namespace.
The Workflow controller will watch the ConfigMap and update the size of the semaphore accordingly, so the Workflow controller must have permission to read the ConfigMap.
If you refer to a common ConfigMap by setting the `namespace` field you can share locks between namespaces.

For local mutexes and all database locks the namespace defaults to the namespace of the Workflow, but is otherwise not actually used in any other way, so the kubernetes namespace doesn't need to exist.
For database locks you can have global locks between all controllers.
For example using overriding the namespace to `global` and using the same lock name in several Workflows in several controllers will use the same lock.

## Understanding Database Lock Keys

Lock keys are used to uniquely identify locks in the system. The format depends on the lock type and where it's used.

For database locks:

- Semaphores use the format `sem/<namespace>/<key>`
- Mutexes use the format `mtx/<namespace>/<key>`

For ConfigMap locks:

- The key used is the value specified in the `key` field in the ConfigMap reference
- The namespace is derived from the Workflow or can be overridden with the `namespace` field

How a lock appears in each table:

- In the limit table: `<namespace>/<key>`
- In the state and lock tables: `sem/<namespace>/<key>` or `mtx/<namespace>/<key>`

When creating database semaphores, always insert records into the limit table with the `<namespace>/<key>` format, not the `sem/` or `mtx/` prefixed format.

## Queuing

When a Workflow cannot acquire a lock it will be placed into a ordered queue.

You can set a [`priority`](parallelism.md#priority) on Workflows.
The queue is first ordered by priority: a higher priority number is placed before a lower priority number.
The queue is then ordered by creation time-stamp: older Workflows are placed before newer Workflows.

Workflows can only acquire a lock if they are at the front of the queue for that lock.
This applies to both local and multiple controller locks.

## Multiple locks

> v3.6 and after

You can specify multiple locks in a single Workflow or Template.

```yaml
synchronization:
  mutexes:
    - name: alpha
    - name: beta
  semaphores:
    - configMapKeyRef:
        key: foo
        name: my-config
    - configMapKeyRef:
        key: bar
        name: my-config
```

The Workflow will block until all of these locks are available.

!!! Warning
    If a Workflow is at the front of the queue and it needs to acquire multiple locks, all other Workflows that also need those same locks will wait. This applies even if the other Workflows only wish to acquire a subset of those locks.

## Workflow-level parallelism

You can use `parallelism` within a Workflow or Template to restrict the total concurrent executions of steps or tasks.
(Note that this only restricts concurrent executions within the same Workflow.)

Examples:

1. [`parallelism-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-limit.yaml) restricts the parallelism of a [loop](walk-through/loops.md)
1. [`parallelism-nested.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-nested.yaml) restricts the parallelism of a nested loop
1. [`parallelism-nested-dag.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-nested-dag.yaml) restricts the number of dag tasks that can be run at any one time
1. [`parallelism-nested-workflow.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-nested-workflow.yaml) shows how parallelism is inherited by children
1. [`parallelism-template-limit.yaml`](https://github.com/argoproj/argo-workflows/blob/main/examples/parallelism-template-limit.yaml) shows how parallelism of looped Templates is also restricted

## Monitoring Lock Status

You can monitor the status of locks in several ways:

1. Through the Workflow status - each Workflow displays its own lock status in `.status.synchronization`

2. By querying the database tables directly for database locks:

   ```sql
   -- View all defined semaphore limits
   SELECT * FROM sync_limit;

   -- See which workflows are currently holding locks
   SELECT * FROM sync_state WHERE held = true;

   -- See which workflows are waiting for locks
   SELECT * FROM sync_state WHERE held = false ORDER BY priority DESC, time ASC;

   -- Check controller health
   SELECT * FROM sync_controller;
   ```

3. For local ConfigMap locks, examine the status in the Workflow resources themselves using kubectl

## Other Parallelism support

You can also [restrict parallelism at the Controller-level](parallelism.md).

## Database configuration

In order to use multiple controller locks you need to configure the database in the workflow-controller-configmap ConfigMap.
This is done by setting up the [`SyncConfig` section](workflow-controller-configmap.md#syncconfig).

If you try to use multiple controller locks without configuring the database you will get an error.

### Limit Table

This table stores the maximum number of concurrent Workflows/Templates allowed for each semaphore.

| Name        | Type      | Description                                                   |
|-------------|-----------|---------------------------------------------------------------|
| `name`      | `string`  | The semaphore name, in the format namespace/name.             |
| `sizelimit` | `integer` | The maximum number of concurrent Workflows/Templates allowed. |

This table is created automatically when the controller starts.
The table name is configured in the workflow-controller-configmap `limitTableName` field, and defaults to `sync_limit`.

You are expected to manually insert values for any semaphores you want to use.

#### Configuring limits

To allow up to 3 concurrent locks for semaphore "namespace/semaphore":

```sql
INSERT INTO sync_limit (name, sizelimit) VALUES ('namespace/semaphore', 3);
```

To change the limit to 4:

```sql
UPDATE sync_limit SET sizelimit = 4 WHERE name = 'namespace/semaphore';
```

### Heartbeat Table

This table stores the last heartbeat time-stamp for each controller.

| Name         | Type         | Description                                                                         |
|--------------|--------------|-------------------------------------------------------------------------------------|
| `controller` | `string`     | The controller name, from the workflow-controller-configmap `controllerName` field. |
| `time`       | `time-stamp` | The last heartbeat time-stamp.                                                      |

This table is created automatically when the controller starts.
The table name is configured in the workflow-controller-configmap `controllerTableName` field, and defaults to `sync_controller`.

Each controller maintains a heartbeat in the database to indicate it is active. If a controller stops updating its heartbeat, it is considered inactive.
Workflows from other controllers can then acquire locks that were previously held by the inactive controller.

This prevents locks from being permanently blocked by inactive controllers.
Pending locks are only considered as being in the queue if the controller has updated its heartbeat within the last `inactiveControllerSeconds` seconds.

#### Heartbeat defaults

- Heartbeat Interval: Every 60 seconds. `heartbeatSeconds`
- Inactive Controller Timeout: 300 seconds without a heartbeat update. `inactiveControllerSeconds`

These values can be configured in the workflow-controller-configmap ConfigMap.

The `inactiveControllerSeconds` value also applies to entries in the Lock table, determining when locks from inactive controllers are considered stale and can be bypassed. These entries are just used whilst acquiring the lock along with a database transaction for the state.

Held locks are never taken by another Workflow, even if the controller is inactive.
You must manually intervene to release a held lock from a inactive controller.

### State Table

This table stores the current state of each mutex/semaphore.

| Name          | Type        | Description                                                                     |
|---------------|-------------|---------------------------------------------------------------------------------|
| `name`        | `string`    | The semaphore name, in the format `sem/namespace/lock` or `mtx/namespace/lock`. |
| `workflowkey` | `string`    | The key of the Workflow that is holding or waiting for the lock.                |
| `controller`  | `string`    | The controller name as configured by `controllerName`.                          |
| `held`        | `boolean`   | Indicates whether the semaphore is currently held (true) or pending (false).    |
| `priority`    | `integer`   | The priority of the Workflow (higher number = higher priority).                 |
| `time`        | `timestamp` | The creation time-stamp of the workflow.                                        |

This table is created automatically when the controller starts.
The table name is configured in the workflow-controller-configmap `stateTableName` field, and defaults to `sync_state`.

This table maintains the state of each semaphore or mutex, including details about the Workflow holding the lock.
Manual modifications to this table are not recommended except to release a held lock from a inactive controller.

This is polled by the controller every `pollSeconds` seconds.

### Lock Table

This table stores the lock information for coordination between controllers.

| Name         | Type        | Description                                                                      |
|--------------|-------------|----------------------------------------------------------------------------------|
| `name`       | `string`    | The semaphore name, in the form of `sem/namespace/lock` or `mtx/namespace/lock`. |
| `controller` | `string`    | The controller name as configured by `controllerName`.                           |
| `time`       | `timestamp` | The time-stamp of lock creation.                                                 |

This table is created automatically when the controller starts.
The table name is configured in the workflow-controller-configmap `lockTableName` field, and defaults to `sync_lock`.

This table is used internally for coordination between multiple controllers to avoid race conditions when acquiring locks.
It does not store the active state of a lock - that is in the state table.
Manual modifications to this table are not recommended except to release a lock from an inactive controller.

### Troubleshooting Database Lock Issues

If Workflows are not being processed as expected:

1. Check if a lock is being held by an inactive controller:

   ```sql
   SELECT s.name, s.workflowkey, s.controller, c.time as last_heartbeat
   FROM sync_state s
   LEFT JOIN sync_controller c ON s.controller = c.controller
   WHERE s.held = true;
   ```

2. Verify semaphore limits are correctly set:

   ```sql
   SELECT * FROM sync_limit WHERE name = 'namespace/lock-name';
   ```

3. If a workflow seems stuck waiting for a lock, check its position in the queue:

   ```sql
   SELECT workflowkey, priority, time
   FROM sync_state
   WHERE name = 'sem/namespace/lock-name' AND held = false
   ORDER BY priority DESC, time ASC;
   ```

#### Permanent controller removal

To remove a deleted controller permanently, you can use the following SQL statements:

```sql
DELETE FROM sync_controller WHERE controller = 'inactive-controller';
DELETE FROM sync_state WHERE controller = 'inactive-controller';
```
