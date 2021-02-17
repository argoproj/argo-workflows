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
| `DEFAULT_REQUEUE_TIME` | `time.Duration` | The requeue time for the rate limiter of the workflow queue. |
| `GZIP_IMPLEMENTATION` | `string` | The implementation of compression/decompression. Currently only "PGZip" and "GZip" are supported. Defaults to "PGZip". |
| `LEADER_ELECTION_IDENTITY` | `string` | The ID used for workflow controllers to elect a leader. |
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

## Executor

| Name | Type | Description|
|----------|------|------------|
| `ARGO_CONTAINER_RUNTIME_EXECUTOR` | `string` | The name of the container runtime executor. |
| `ARGO_KUBELET_PORT` | `int` | The port to the Kubelet API. |
| `ARGO_KUBELET_INSECURE` | `bool` | Whether to disable the TLS verification. |
| `PNS_PRIVILEGED` | `bool` | Whether to always set privileged on for PNS when PNS executor is used. |
| `REMOVE_LOCAL_ART_PATH` | `bool` | Whether to remove local artifacts. |
