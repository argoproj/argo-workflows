# Environment Variables

This document outlines environment variables that can be used to customize behavior.

!!! Warning
    Environment variables are typically added to test out experimental features and should not be used by most users.
    Environment variables may be removed at any time.

## Controller

| Name                                     | Type                | Default                                                                                     | Description                                                                                                                                                                                                                                                              |
|------------------------------------------|---------------------|---------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ARGO_AGENT_TASK_WORKERS`                | `int`               | `16`                                                                                        | The number of task workers for the agent pod.                                                                                                                                                                                                                            |
| `ALL_POD_CHANGES_SIGNIFICANT`            | `bool`              | `false`                                                                                     | Whether to consider all pod changes as significant during pod reconciliation.                                                                                                                                                                                            |
| `ALWAYS_OFFLOAD_NODE_STATUS`             | `bool`              | `false`                                                                                     | Whether to always offload the node status.                                                                                                                                                                                                                               |
| `ARCHIVED_WORKFLOW_GC_PERIOD`            | `time.Duration`     | `24h`                                                                                       | The periodicity for GC of archived workflows.                                                                                                                                                                                                                            |
| `ARGO_PPROF`                             | `bool`              | `false`                                                                                     | Enable [`pprof`](https://go.dev/blog/pprof) endpoints                                                                                                                                                                                                                                                 |
| `ARGO_PROGRESS_PATCH_TICK_DURATION`      | `time.Duration`     | `1m`                                                                                        | How often self reported progress is patched into the pod annotations which means how long it takes until the controller picks up the progress change. Set to 0 to disable self reporting progress.                                                                       |
| `ARGO_PROGRESS_FILE_TICK_DURATION`       | `time.Duration`     | `3s`                                                                                        | How often the progress file is read by the executor. Set to 0 to disable self reporting progress.                                                                                                                                                                        |
| `ARGO_REMOVE_PVC_PROTECTION_FINALIZER`   | `bool`              | `true`                                                                                      | Remove the `kubernetes.io/pvc-protection` finalizer from persistent volume claims (PVC) after marking PVCs created for the workflow for deletion, so deleted is not blocked until the pods are deleted.  [#6629](https://github.com/argoproj/argo-workflows/issues/6629) |
| `ARGO_TRACE`                             | `string`            | ``                                                                                          | Whether to enable tracing statements in Argo components.                                                                                                                                                                                                                 |
| `ARGO_AGENT_PATCH_RATE`                  | `time.Duration`     | `DEFAULT_REQUEUE_TIME`                                                                      | Rate that the Argo Agent will patch the workflow task-set.                                                                                                                                                                                                               |
| `ARGO_AGENT_CPU_LIMIT`                   | `resource.Quantity` | `100m`                                                                                      | CPU resource limit for the agent.                                                                                                                                                                                                                                        |
| `ARGO_AGENT_MEMORY_LIMIT`                | `resource.Quantity` | `256m`                                                                                      | Memory resource limit for the agent.                                                                                                                                                                                                                                     |
| `ARGO_POD_STATUS_CAPTURE_FINALIZER`      | `bool`              | `false`                                                                                     | The finalizer blocks the deletion of pods until the controller captures their status.
| `BUBBLE_ENTRY_TEMPLATE_ERR`              | `bool`              | `true`                                                                                      | Whether to bubble up template errors to workflow.                                                                                                                                                                                                                        |
| `CACHE_GC_PERIOD`                        | `time.Duration`     | `0s`                                                                                        | How often to perform memoization cache GC, which is disabled by default and can be enabled by providing a non-zero duration.                                                                                                                                             |
| `CACHE_GC_AFTER_NOT_HIT_DURATION`        | `time.Duration`     | `30s`                                                                                       | When a memoization cache has not been hit after this duration, it will be deleted.                                                                                                                                                                                       |
| `CRON_SYNC_PERIOD`                       | `time.Duration`     | `10s`                                                                                       | How often to sync cron workflows.                                                                                                                                                                                                                                        |
| `DEFAULT_REQUEUE_TIME`                   | `time.Duration`     | `10s`                                                                                       | The re-queue time for the rate limiter of the workflow queue.                                                                                                                                                                                                            |
| `DISABLE_MAX_RECURSION`                  | `bool`              | `false`                                                                                     | Set to true to disable the recursion preventer, which will stop a workflow running which has called into a child template 100 times                                                                                                                                      |
| `EXPRESSION_TEMPLATES`                   | `bool`              | `true`                                                                                      | Escape hatch to disable expression templates.                                                                                                                                                                                                                            |
| `EVENT_AGGREGATION_WITH_ANNOTATIONS`     | `bool`              | `false`                                                                                     | Whether event annotations will be used when aggregating events.                                                                                                                                                                                                          |
| `GZIP_IMPLEMENTATION`                    | `string`            | `PGZip`                                                                                     | The implementation of compression/decompression. Currently only "`PGZip`" and "`GZip`" are supported.                                                                                                                                                                    |
| `INFORMER_WRITE_BACK`                    | `bool`              | `true`                                                                                      | Whether to write back to informer instead of catching up.                                                                                                                                                                                                                |
| `HEALTHZ_AGE`                            | `time.Duration`     | `5m`                                                                                        | How old a un-reconciled workflow is to report unhealthy.                                                                                                                                                                                                                 |
| `INDEX_WORKFLOW_SEMAPHORE_KEYS`          | `bool`              | `true`                                                                                      | Whether or not to index semaphores.                                                                                                                                                                                                                                      |
| `LEADER_ELECTION_IDENTITY`               | `string`            | Controller's `metadata.name`                                                                | The ID used for workflow controllers to elect a leader.                                                                                                                                                                                                                  |
| `LEADER_ELECTION_DISABLE`                | `bool`              | `false`                                                                                     | Whether leader election should be disabled.                                                                                                                                                                                                                              |
| `LEADER_ELECTION_LEASE_DURATION`         | `time.Duration`     | `15s`                                                                                       | The duration that non-leader candidates will wait to force acquire leadership.                                                                                                                                                                                           |
| `LEADER_ELECTION_RENEW_DEADLINE`         | `time.Duration`     | `10s`                                                                                       | The duration that the acting master will retry refreshing leadership before giving up.                                                                                                                                                                                   |
| `LEADER_ELECTION_RETRY_PERIOD`           | `time.Duration`     | `5s`                                                                                        | The duration that the leader election clients should wait between tries of actions.                                                                                                                                                                                      |
| `MAX_OPERATION_TIME`                     | `time.Duration`     | `30s`                                                                                       | The maximum time a workflow operation is allowed to run for before re-queuing the workflow onto the work queue.                                                                                                                                                          |
| `OFFLOAD_NODE_STATUS_TTL`                | `time.Duration`     | `5m`                                                                                        | The TTL to delete the offloaded node status. Currently only used for testing.                                                                                                                                                                                            |
| `OPERATION_DURATION_METRIC_BUCKET_COUNT` | `int`               | `6`                                                                                         | The number of buckets to collect the metric for the operation duration.                                                                                                                                                                                                  |
| `POD_NAMES`                              | `string`            | `v2`                                                                                        | Whether to have pod names contain the template name (v2) or be the node id (v1) - should be set the same for Argo Server.                                                                                                                                                |
| `RECENTLY_STARTED_POD_DURATION`          | `time.Duration`     | `10s`                                                                                       | The duration of a pod before the pod is considered to be recently started.                                                                                                                                                                                               |
| `RECENTLY_DELETED_POD_DURATION`          | `time.Duration`     | `2m`                                                                                       | The duration of a pod before the pod is considered to be recently deleted.                                                                                                                                                                                               |
| `RETRY_BACKOFF_DURATION`                 | `time.Duration`     | `10ms`                                                                                      | The retry back-off duration when retrying API calls.                                                                                                                                                                                                                     |
| `RETRY_BACKOFF_FACTOR`                   | `float`             | `2.0`                                                                                       | The retry back-off factor when retrying API calls.                                                                                                                                                                                                                       |
| `RETRY_BACKOFF_STEPS`                    | `int`               | `5`                                                                                         | The retry back-off steps when retrying API calls.                                                                                                                                                                                                                        |
| `RETRY_HOST_NAME_LABEL_KEY`              | `string`            | `kubernetes.io/hostname`                                                                    | The label key for host name used when retrying templates.                                                                                                                                                                                                                |
| `TRANSIENT_ERROR_PATTERN`                | `string`            | `""`                                                                                        | The regular expression that represents additional patterns for transient errors.                                                                                                                                                                                         |
| `WF_DEL_PROPAGATION_POLICY`              | `string`            | `""`                                                                                        | The deletion propagation policy for workflows.                                                                                                                                                                                                                           |
| `WORKFLOW_GC_PERIOD`                     | `time.Duration`     | `5m`                                                                                        | The periodicity for GC of workflows.                                                                                                                                                                                                                                     |
| `SEMAPHORE_NOTIFY_DELAY`                 | `time.Duration`     | `1s`                                                                                        | Tuning Delay when notifying semaphore waiters about availability in the semaphore                                                                                                                                                                                        |
| `WATCH_CONTROLLER_SEMAPHORE_CONFIGMAPS` | `bool` | `true` | Whether to watch the Controller's ConfigMap and semaphore ConfigMaps for run-time changes. When disabled, the Controller will only read these ConfigMaps once and will have to be manually restarted to pick up new changes. |
| `SKIP_WORKFLOW_DURATION_ESTIMATION` | `bool` | `false` | Whether to lookup resource usage from prior workflows to estimate usage for new workflows. |

CLI parameters of the Controller can be specified as environment variables with the `ARGO_` prefix.
For example:

```bash
workflow-controller --managed-namespace=argo
```

Can be expressed as:

```bash
ARGO_MANAGED_NAMESPACE=argo workflow-controller
```

You can set environment variables for the Controller Deployment's container spec like the following:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-controller
spec:
  selector:
    matchLabels:
      app: workflow-controller
  template:
    metadata:
      labels:
        app: workflow-controller
    spec:
      containers:
        - env:
            - name: WORKFLOW_GC_PERIOD
              value: 30s
```

## Executor

| Name                                   | Type            | Default | Description                                                                                            |
|----------------------------------------|-----------------|---------|--------------------------------------------------------------------------------------------------------|
| `ARGO_DEBUG_PAUSE_AFTER`               | `bool`          | `false` | Enable [Debug Pause](debug-pause.md) after step execution
| `ARGO_DEBUG_PAUSE_BEFORE`              | `bool`          | `false` | Enable [Debug Pause](debug-pause.md) before step execution
| `EXECUTOR_RETRY_BACKOFF_DURATION`      | `time.Duration` | `1s`    | The retry back-off duration when the workflow executor performs retries.                               |
| `EXECUTOR_RETRY_BACKOFF_FACTOR`        | `float`         | `1.6`   | The retry back-off factor when the workflow executor performs retries.                                 |
| `EXECUTOR_RETRY_BACKOFF_JITTER`        | `float`         | `0.5`   | The retry back-off jitter when the workflow executor performs retries.                                 |
| `EXECUTOR_RETRY_BACKOFF_STEPS`         | `int`           | `5`     | The retry back-off steps when the workflow executor performs retries.                                  |
| `REMOVE_LOCAL_ART_PATH`                | `bool`          | `false` | Whether to remove local artifacts.                                                                     |
| `RESOURCE_STATE_CHECK_INTERVAL`        | `time.Duration` | `5s`    | The time interval between resource status checks against the specified success and failure conditions. |
| `WAIT_CONTAINER_STATUS_CHECK_INTERVAL` | `time.Duration` | `5s`    | The time interval for wait container to check whether the containers have completed.                   |

You can set environment variables for the Executor in your [`workflow-controller-configmap`](workflow-controller-configmap.md) like the following:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
data:
  config: |
    executor:
      env:
      - name: RESOURCE_STATE_CHECK_INTERVAL
        value: 3s
```

## Argo Server

| Name                                       | Type     | Default | Description                                                                                                             |
|--------------------------------------------|----------|---------|-------------------------------------------------------------------------------------------------------------------------|
| `ARGO_ARTIFACT_SERVER`                     | `bool`   | `true`  | Enable [Workflow Archive](workflow-archive.md) endpoints
| `ARGO_PPROF`                               | `bool`   | `false` | Enable [`pprof`](https://go.dev/blog/pprof) endpoints
| `ARGO_SERVER_METRICS_AUTH`                 | `bool`   | `true`  | Enable auth on the `/metrics` endpoint
| `DISABLE_VALUE_LIST_RETRIEVAL_KEY_PATTERN` | `string` | `""`    | Disable the retrieval of the list of label values for keys based on this regular expression.                            |
| `FIRST_TIME_USER_MODAL`                    | `bool`   | `true`  | Show this modal.                                                                                                        |
| `FEEDBACK_MODAL`                           | `bool`   | `true`  | Show this modal.                                                                                                        |
| `GRPC_MESSAGE_SIZE`                        | `string` | `104857600` | Use different GRPC Max message size for Server (supporting huge workflows).                                         |
| `IP_KEY_FUNC_HEADERS`                      | `string` | `""`    | List of comma separated request headers containing IPs to use for rate limiting. For example, "X-Forwarded-For,X-Real-IP". By default, uses the request's remote IP address.          |
| `NEW_VERSION_MODAL`                        | `bool`   | `true`  | Show this modal.                                                                                                        |
| `POD_NAMES`                                | `string` | `v2`    | Whether to have pod names contain the template name (v2) or be the node id (v1) - should be set the same for Controller |
| `SSO_DELEGATE_RBAC_TO_NAMESPACE`           | `bool`   | `false` | Enable [SSO RBAC Namespace Delegation](argo-server-sso.md#sso-rbac-namespace-delegation)

CLI parameters of the Server can be specified as environment variables with the `ARGO_` prefix.
For example:

```bash
argo server --managed-namespace=argo
```

Can be expressed as:

```bash
ARGO_MANAGED_NAMESPACE=argo argo server
```

You can set environment variables for the Server Deployment's container spec like the following:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: argo-server
spec:
  selector:
    matchLabels:
      app: argo-server
  template:
    metadata:
      labels:
        app: argo-server
    spec:
      containers:
        - args:
            - server
          image: quay.io/argoproj/argocli:latest
          name: argo-server
          env:
            - name: GRPC_MESSAGE_SIZE
              value: "209715200"
          ports:
          # ...
```
