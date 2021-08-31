# Environment Variables

This document outlines the set of environment variables that can be used to customize the behaviours at different levels.
These environment variables are typically added to test out experimental features and should not be needed by most users.
Note that these environment variables may be removed at any time.

## Controller

| Name | Type | Default | Description | 
|------|------|---------|-------------|
| `ALL_POD_CHANGES_SIGNIFICANT` | `bool` | `false` | Whether to consider all pod changes as significant during pod reconciliation. |
| `ALWAYS_OFFLOAD_NODE_STATUS` | `bool` | `false` | Whether to always offload the node status. |
| `ARCHIVED_WORKFLOW_GC_PERIOD` | `time.Duration` | `24h` | The periodicity for GC of archived workflows. |
| `ARGO_PPROF` | `bool` | `false` | Enable pprof endpoints |
| `ARGO_TRACE` | `string` | `"1"` | Whether to enable tracing statements in Argo components. |
| `CRON_SYNC_PERIOD` | `time.Duration` | `10s` | How often to sync cron workflows. |
| `DEFAULT_REQUEUE_TIME` | `time.Duration` | `10s` | The requeue time for the rate limiter of the workflow queue. |
| `EXPRESSION_TEMPLATES` | `bool` | `true` | Escape hatch to disable expression templates. |
| `GZIP_IMPLEMENTATION` | `string` | `"PGZip"` | The implementation of compression/decompression. Currently only "PGZip" and "GZip" are supported. |
| `HEALTHZ_AGE` | `time.Duration` | `5m` | How old a un-reconciled workflow is to report unhealthy. |
| `INDEX_WORKFLOW_SEMAPHORE_KEYS` | `bool` | `true` | Whether or not to index semaphores. |
| `LEADER_ELECTION_IDENTITY` | `string` | Controller's `metadata.name` | The ID used for workflow controllers to elect a leader. |
| `LEADER_ELECTION_DISABLE` | `bool` | `false` | Whether leader election should be disabled. |
| `LEADER_ELECTION_LEASE_DURATION` | `time.Duration` | `15s` | The duration that non-leader candidates will wait to force acquire leadership. |
| `LEADER_ELECTION_RENEW_DEADLINE` | `time.Duration` | `10s` | The duration that the acting master will retry refreshing leadership before giving up. |
| `LEADER_ELECTION_RETRY_PERIOD` | `time.Duration` | `5s` | The duration that the leader election clients should wait between tries of actions. |
| `MAX_OPERATION_TIME` | `time.Duration` | `30s` | The maximum time a workflow operation is allowed to run for before requeuing the workflow onto the work queue. |
| `OFFLOAD_NODE_STATUS_TTL` | `time.Duration` | `5m` | The TTL to delete the offloaded node status. Currently only used for testing. |
| `POD_NAMES` | `string` | `v2` | Whether to have pod names contain the template name (v2) or be the node id (v1). |
| `RECENTLY_STARTED_POD_DURATION` | `time.Duration` | `10s` | The duration of a pod before the pod is considered to be recently started. |
| `RETRY_BACKOFF_DURATION` | `time.Duration` | `10ms` | The retry backoff duration when retrying API calls. |
| `RETRY_BACKOFF_FACTOR` | `float` | `2.0` | The retry backoff factor when retrying API calls. |
| `RETRY_BACKOFF_STEPS` | `int` | `5` | The retry backoff steps when retrying API calls. |
| `RETRY_HOST_NAME_LABEL_KEY` | `string` | `kubernetes.io/hostname` | The label key for host name used when retrying templates. |
| `TRANSIENT_ERROR_PATTERN` | `string` | `""` | The regular expression that represents additional patterns for transient errors. |
| `WF_DEL_PROPAGATION_POLICY` | `string` | `""` | The deletion propagation policy for workflows. |
| `WORKFLOW_GC_PERIOD` | `time.Duration` | `5m` | The periodicity for GC of workflows. |
| `BUBBLE_ENTRY_TEMPLATE_ERR` | `bool` | `true` | Whether to bubble up template errors to workflow. |
| `INFORMER_WRITE_BACK` | `bool` | `true` | Whether to write back to informer instead of catching up. |
| `GRPC_MESSAGE_SIZE` | `string` | Use different GRPC Max message size for Argo server deployment (supporting huge workflows). |
| `ARGO_PROGRESS_PATCH_TICK_DURATION` | `time.Duration` | `1m` | How often self reported progress is patched into the pod annotations which means how long it takes until the controller picks up the progress change. Set to 0 to disable self reporting progress. |
| `ARGO_PROGRESS_FILE_TICK_DURATION` | `time.Duration` | `3s` | How often the progress file is read by the executor. Set to 0 to disable self reporting progress. |

CLI parameters of the `argo-server` and `workflow-controller` can be specified as environment variables with the `ARGO_` prefix. For example:

```
workflow-controller --managed-namespace=argo
```

Can be expressed as:

```
ARGO_MANAGED_NAMESPACE=argo workflow-controller
```

You can set environment variable for the argo-server deployment, for example:

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
        image: argoproj/argocli:latest
        name: argo-server
        env:
        - name: GRPC_MESSAGE_SIZE
          value: "209715200"
        ports:
        ..
        ...
        ....
```

You can set the environment variables for controller in controller's container spec like the following:

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

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `ARGO_CONTAINER_RUNTIME_EXECUTOR` | `string` | `"docker"` | The name of the container runtime executor. |
| `ARGO_KUBELET_PORT` | `int` | `10250` | The port to the Kubelet API. |
| `ARGO_KUBELET_INSECURE` | `bool` | `false` | Whether to disable the TLS verification. |
| `EXECUTOR_RETRY_BACKOFF_DURATION` | `time.Duration` | `1s` | The retry backoff duration when the workflow executor performs retries. |
| `EXECUTOR_RETRY_BACKOFF_FACTOR` | `float` | `1.6` | The retry backoff factor when the workflow executor performs retries. |
| `EXECUTOR_RETRY_BACKOFF_JITTER` | `float` | `0.5` | The retry backoff jitter when the workflow executor performs retries. |
| `EXECUTOR_RETRY_BACKOFF_STEPS` | `int` | `5` | The retry backoff steps when the workflow executor performs retries. |
| `PNS_PRIVILEGED` | `bool` | `false` | Whether to always set privileged on for PNS when PNS executor is used. |
| `REMOVE_LOCAL_ART_PATH` | `bool` | `false` | Whether to remove local artifacts. |
| `RESOURCE_STATE_CHECK_INTERVAL` | `time.Duration` | `5s` | The time interval between resource status checks against the specified success and failure conditions. |
| `WAIT_CONTAINER_STATUS_CHECK_INTERVAL` | `time.Duration` | `5s` | The time interval for wait container to check whether the containers have completed. |

You can set the environment variables for executor by customizing executor container's environment variables in your
controller's configmap like the following:

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
