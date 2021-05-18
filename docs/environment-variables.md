# Environment Variables

This document outlines the set of environment variables that can be used to customize the behaviours at different levels.
These environment variables are typically added to test out experimental features and should not be needed by most users.
Note that these environment variables may be removed at any time.

## Controller

| Name | Type | Description|
|----------|------|------------|
| `ALL_POD_CHANGES_SIGNIFICANT` | `bool` |  Whether to consider all pod changes as significant during pod reconciliation. |
| `ALWAYS_OFFLOAD_NODE_STATUS` | `bool` | Whether to always offload the node status. |
| `ARCHIVED_WORKFLOW_GC_PERIOD` | `time.Duration` | The periodicity for GC of archived workflows. |
| `ARGO_TRACE` | `bool` | Whether to enable tracing statements in Argo components. |
| `CRON_SYNC_PERIOD` | `time.Duration` | How ofen to sync cron workflows. Default `10s` |
| `DEFAULT_REQUEUE_TIME` | `time.Duration` | The requeue time for the rate limiter of the workflow queue. |
| `EXPRESSION_TEMPLATES` | `bool` | Escape hatch to disable expression templates. Default `true`. |
| `GZIP_IMPLEMENTATION` | `string` | The implementation of compression/decompression. Currently only "PGZip" and "GZip" are supported. Defaults to "PGZip". |
| `INDEX_WORKFLOW_SEMAPHORE_KEYS` | `bool` | Whether or not to index semaphores. Defaults to `true`. |
| `LEADER_ELECTION_IDENTITY` | `string` | The ID used for workflow controllers to elect a leader. |
| `LEADER_ELECTION_DISABLE` | `bool` | Whether leader election should be disabled. |
| `LEADER_ELECTION_LEASE_DURATION` | `time.Duration` | The duration that non-leader candidates will wait to force acquire leadership. |
| `LEADER_ELECTION_RENEW_DEADLINE` | `time.Duration` | The duration that the acting master will retry refreshing leadership before giving up. |
| `LEADER_ELECTION_RETRY_PERIOD` | `time.Duration` | The duration that the leader election clients should wait between tries of actions. |
| `MAX_OPERATION_TIME` | `time.Duration` | The maximum time a workflow operation is allowed to run for before requeuing the workflow onto the work queue. |
| `OFFLOAD_NODE_STATUS_TTL` | `time.Duration` | The TTL to delete the offloaded node status. Currently only used for testing. |
| `RECENTLY_STARTED_POD_DURATION` | `time.Duration` | The duration of a pod before the pod is considered to be recently started. |
| `RETRY_BACKOFF_DURATION` | `time.Duration` | The retry backoff duration when retrying API calls. |
| `RETRY_BACKOFF_FACTOR` | `float` | The retry backoff factor when retrying API calls. |
| `RETRY_BACKOFF_STEPS` | `int` | The retry backoff steps when retrying API calls. |
| `TRANSIENT_ERROR_PATTERN` | `string` | The regular expression that represents additional patterns for transient errors. |
| `WF_DEL_PROPAGATION_POLICY` | `string` | The deletion propogation policy for workflows. |
| `WORKFLOW_GC_PERIOD` | `time.Duration` | The periodicity for GC of workflows. |
| `BUBBLE_ENTRY_TEMPLATE_ERR` | `bool` | Whether to bubble up template errors to workflow. Default true |
| `INFORMER_WRITE_BACK` | `bool` | Whether to write back to informer instead of catching up. Deafult true |

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

| Name | Type | Description|
|----------|------|------------|
| `ARGO_CONTAINER_RUNTIME_EXECUTOR` | `string` | The name of the container runtime executor. |
| `ARGO_KUBELET_PORT` | `int` | The port to the Kubelet API. |
| `ARGO_KUBELET_INSECURE` | `bool` | Whether to disable the TLS verification. |
| `EXECUTOR_RETRY_BACKOFF_DURATION` | `time.Duration` | The retry backoff duration when the workflow executor performs retries. |
| `EXECUTOR_RETRY_BACKOFF_FACTOR` | `float` | The retry backoff factor when the workflow executor performs retries. |
| `EXECUTOR_RETRY_BACKOFF_JITTER` | `float` | The retry backoff jitter when the workflow executor performs retries. |
| `EXECUTOR_RETRY_BACKOFF_STEPS` | `int` | The retry backoff steps when the workflow executor performs retries. |
| `PNS_PRIVILEGED` | `bool` | Whether to always set privileged on for PNS when PNS executor is used. |
| `REMOVE_LOCAL_ART_PATH` | `bool` | Whether to remove local artifacts. |
| `RESOURCE_STATE_CHECK_INTERVAL` | `time.Duration` | The time interval between resource status checks against the specified success and failure conditions. |

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
