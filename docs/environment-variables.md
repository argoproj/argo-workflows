# Environment Variables

This document outlines the set of environment variables that can be used to customize the behaviours at different levels.
Note that these environment variables are not officially supported and may be removed at any time.

## Controller

| Name | Type | Description|
|----------|------|------------|
| `ALL_POD_CHANGES_SIGNIFICANT` | bool |  Whether to consider all pod changes as significant during pod reconciliation. |
| `ALWAYS_OFFLOAD_NODE_STATUS` | bool | Whether to always offload the node status. |
| `ARCHIVED_WORKFLOW_GC_PERIOD` | time.duration | The periodicity for GC of archived workflows. |
| `ARGO_TRACE` | bool | Whether to enable tracing statements in Argo components. |
| `DEFAULT_REQUEUE_TIME` | time.duration | The requeue time for the rate limiter of the workflow queue. |
| `LEADER_ELECTION_IDENTITY` | string | The ID used for workflow controllers to elect a leader. |
| `MAX_OPERATION_TIME` | time.duration | The maximum time a workflow operation is allowed to run for before requeuing the workflow onto the work queue. |
| `OFFLOAD_NODE_STATUS_TTL` | time.duration | The TTL to delete the offloaded node status. Currently only used for testing. |
| `POD_GC_TRANSIENT_ERROR_MAX_REQUEUES` | int | The maximum number of requeues of pods in the GC queue when encountering transient errors. |
| `RECENTLY_STARTED_POD_DURATION` | time.duration | The duration of a pod before the pod is considered to be recently started. |
| `RETRY_BACKOFF_DURATION` | time.duration | The retry backoff duration when retrying API calls. |
| `RETRY_BACKOFF_FACTOR` | float | The retry backoff factor when retrying API calls. |
| `RETRY_BACKOFF_STEPS` | int | The retry backoff steps when retrying API calls. |
| `TRANSIENT_ERROR_PATTERN` | string | The regular expression that represents additional patterns for transient errors. |
| `WF_DEL_PROPAGATION_POLICY` | string | The deletion propogation policy for workflows. |
| `WORKFLOW_GC_PERIOD` | time.duration | The periodicity for GC of workflows. |


## Executor

| Name | Type | Description|
|----------|------|------------|
| `ARGO_CONTAINER_RUNTIME_EXECUTOR` | string | The name of the container runtime executor. |
| `ARGO_KUBELET_PORT` | int | The port to the Kubelet API. |
| `ARGO_KUBELET_INSECURE` | bool | Whether to disable the TLS verification. |
| `PNS_PRIVILEGED` | bool | Whether to always set privileged on for PNS when PNS executor is used. |
| `REMOVE_LOCAL_ART_PATH` | bool | Whether to remove local artifacts. |


## CLI

| Name | Type | Description|
|----------|------|------------|
| `ARGO_INSTANCEID` | string | The controller's instance ID to establish the connection with. |
| `ARGO_NAMESPACE` | string | The namespace to establish the connection with. |
| `ARGO_SERVER` | string | The address of the Argo Server. |
| `ARGO_TOKEN` | string | The authentication token. |
| `BASE_HREF` | string |The base href in Argo Server's index page. Used if the server is running behind reverse proxy under subpath different from "/". |
