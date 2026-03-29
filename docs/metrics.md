# Metrics

## Introduction

Argo emits a certain number of controller metrics that inform on the state of the controller at any given time.
Furthermore, users can also define their own custom metrics to inform on the state of their Workflows.

Custom metrics can be defined to be emitted on a `Workflow`- and `Template`-level basis. These can be useful for many cases; some examples:

- Keeping track of the duration of a `Workflow` or `Template` over time, and setting an alert if it goes beyond a threshold
- Keeping track of the number of times a `Workflow` or `Template` fails over time
- Reporting an important internal metric, such as a model training score or an internal error rate

Emitting custom metrics with Argo is easy, but it's important to understand what makes a good metric and the best way to define metrics in Argo to avoid problems such as [cardinality explosion](https://stackoverflow.com/questions/46373442/how-dangerous-are-high-cardinality-labels-in-prometheus).

Metrics can be collected using the OpenTelemetry protocol or via Prometheus compatible scraping.

## Metrics Configuration

Metrics can be collected via OpenTelemetry protocol or Prometheus scraping.
See [telemetry configuration](telemetry-configuration.md#metrics) for setup details.

## Metrics in Argo

There are two kinds of metrics emitted by Argo: **controller metrics** and **custom metrics**.

### Controller metrics

Metrics that inform on the state of the controller; i.e., they answer the question "What is the state of the controller right now?"
Default controller metrics can be scraped from service ```workflow-controller-metrics``` at the endpoint ```<host>:9090/metrics```

### Custom metrics

Metrics that inform on the state of a Workflow, or a series of Workflows.
These custom metrics are defined by the user in the Workflow spec.

Emitting custom metrics is the responsibility of the emitter owner.
Since the user defines Workflows in Argo, the user is responsible for emitting metrics correctly.

Currently, custom metrics and their labels must be valid Prometheus and OpenTelemetry metric names, which limits them to alphanumeric characters and `_`.
This applies even if you're only using OpenTelemetry for metrics.

### What is and isn't a Prometheus metric

Prometheus metrics should be thought of as ephemeral data points of running processes; i.e., they are the answer to
the question "What is the state of my system _right now_?".
Metrics should report things such as:

- a counter of the number of times a workflow or steps has failed, or
- a gauge of workflow duration, or
- an average of an internal metric such as a model training score or error rate.

Metrics are then routinely scraped and stored and -- when they are correctly designed -- they can represent time series.
Aggregating the examples above over time could answer useful questions such as:

- How has the error rate of this workflow or step changed over time?
- How has the duration of this workflow changed over time? Is the current workflow running for too long?
- Is our model improving over time?

Prometheus metrics should **not** be thought of as a store of data. Since metrics should only report the state of the system at the current time, they should not be used to report historical data such as:

- the status of an individual instance of a workflow, or
- how long a particular instance of a step took to run.

Metrics are also ephemeral, meaning there is no guarantee that they will be persisted for any amount of time.
If you need a way to view and analyze historical data, consider the [workflow archive](workflow-archive.md) or reporting to logs.

### Counter, gauge and histogram

These terms are [defined by OpenTelemetry](https://opentelemetry.io/docs/concepts/signals/metrics/#metric-instruments):

> - `counter`: A value that accumulates over time – you can think of this like an odometer on a car; it only ever goes up.
    - The rate of change of counters can be a very powerful tool in understanding these metrics.
> - `gauge`: Measures a current value at the time it is read. An example would be the fuel gauge in a vehicle.
> - `histogram`: A client-side aggregation of values, such as request latencies. A histogram is a good choice if you are interested in value statistics. For example: How many requests take fewer than 1 second?

### Default Controller Metrics

Metrics for the [Four Golden Signals](https://sre.google/sre-book/monitoring-distributed-systems/#xref_monitoring_golden-signals) are:

- Latency: `queue_latency`
- Traffic: `gauge` and `queue_depth_gauge`
- Errors: `count` and `error_count`
- Saturation: `workers_busy` and `workflow_condition`

!!! Warning "High cardinality"
    Some metric attributes may have high cardinality and are marked with ⚠️ to warn you. You may need to disable this metric or disable the attribute.

<!-- Generated documentation BEGIN -->

#### `client_rate_limiter_latency`

A histogram of the time spent waiting for the client-side rate limiter.
Records the actual wait time spent blocking on the client-go rate limiter before
Kubernetes API requests can proceed. This metric helps identify when the client
rate limiter (configured via QPS and Burst settings) is causing delays in API calls.
This rate limiter is on by default.

This metric has no attributes.

Default bucket sizes: 0.01, 0.1, 0.5, 1, 5, 10, 30, 60, 180

#### `cronworkflows_concurrencypolicy_triggered`

A counter of the number of times a CronWorkflow has triggered its `concurrencyPolicy` to limit the number of workflows running.

|      attribute       |                                   explanation                                    |
|----------------------|----------------------------------------------------------------------------------|
| `name`               | ⚠️ The name of the CronWorkflow                                                   |
| `namespace`          | The namespace that the CronWorkflow is in                                        |
| `concurrency_policy` | The concurrency policy which was triggered, will be either `Forbid` or `Replace` |

#### `cronworkflows_triggered_total`

A counter of the total number of times a CronWorkflow has been triggered.
Suppressed runs due to `concurrencyPolicy: Forbid` will not be counted.

|  attribute  |                explanation                |
|-------------|-------------------------------------------|
| `name`      | ⚠️ The name of the CronWorkflow            |
| `namespace` | The namespace that the CronWorkflow is in |

#### `deprecated_feature`

Incidents of deprecated feature being used.
Deprecated features are [explained here](deprecations.md).
🚨 This counter may go up much more than once for a single use of the feature.

|       attribute        |              explanation              |
|------------------------|---------------------------------------|
| `feature`              | The name of the feature used          |
| `namespace` (optional) | The namespace that the Workflow is in |

`feature` will be one of:

- [`cronworkflow schedule`](deprecations.md#cronworkflow-schedule)
- [`synchronization mutex`](deprecations.md#synchronization-mutex)
- [`synchronization semaphore`](deprecations.md#synchronization-semaphore)
- [`workflow podpriority`](deprecations.md#workflow-podpriority)

#### `error_count`

A counter of certain errors incurred by the controller by cause.

| attribute |      explanation       |
|-----------|------------------------|
| `cause`   | The cause of the error |

The currently tracked specific errors are

- `OperationPanic` - the controller called `panic()` on encountering a programming bug
- `CronWorkflowSubmissionError` - A CronWorkflow failed submission
- `CronWorkflowSpecError` - A CronWorkflow has an invalid specification

#### `gauge`

A gauge of the number of workflows currently in the cluster in each phase.
The `Running` count does not mean that a workflows pods are running, just that the controller has scheduled them.
A workflow can be stuck in `Running` with pending pods for a long time.

| attribute |               explanation               |
|-----------|-----------------------------------------|
| `phase`   | The phase that the Workflow has entered |

#### `is_leader`

Emits 1 if leader, 0 otherwise. Always 1 if leader election is disabled.
A gauge indicating if this Controller is the [leader](high-availability.md#workflow-controller).

- `1` if leader or in standalone mode via [`LEADER_ELECTION_DISABLE=true`](environment-variables.md#controller).
- `0` otherwise, indicating that this controller is a standby that is not currently running workflows.

This metric has no attributes.

#### `k8s_request_duration`

A histogram recording the API requests sent to the Kubernetes API.

|   attribute   |                            explanation                             |
|---------------|--------------------------------------------------------------------|
| `kind`        | The kubernetes `kind` involved in the request such as `configmaps` |
| `verb`        | The verb of the request, such as `Get` or `List`                   |
| `status_code` | The HTTP status code of the response                               |

Default bucket sizes: 0.1, 0.2, 0.5, 1, 2, 5, 10, 20, 60, 180

This contains all the information contained in `k8s_request_total` along with timings.

#### `k8s_request_total`

A counter of the number of API requests sent to the Kubernetes API.

|   attribute   |                            explanation                             |
|---------------|--------------------------------------------------------------------|
| `kind`        | The kubernetes `kind` involved in the request such as `configmaps` |
| `verb`        | The verb of the request, such as `Get` or `List`                   |
| `status_code` | The HTTP status code of the response                               |

This metric is calculable from `k8s_request_duration`, and it is suggested you just collect that metric instead.

#### `log_messages`

A count of log messages emitted by the controller by log level: `error`, `warn` and `info`.

| attribute |         explanation          |
|-----------|------------------------------|
| `level`   | The log level of the message |

#### `operation_duration_seconds`

A histogram of durations of operations.
An operation is a single workflow reconciliation loop within the workflow-controller.
It's the time for the controller to process a single workflow after it has been read from the cluster and is a measure of the performance of the controller affected by the complexity of the workflow.

This metric has no attributes.

The environment variables `OPERATION_DURATION_METRIC_BUCKET_COUNT` and `MAX_OPERATION_TIME` configure the bucket sizes for this metric, unless they are specified using an `histogramBuckets` modifier in the `metricsConfig` block.

#### `pod_missing`

Incidents of pod missing.
A counter of pods that were not seen - for example they are by being deleted by Kubernetes.
You should only see this under high load.

|     attribute      |              explanation               |
|--------------------|----------------------------------------|
| `node_phase`       | The phase that the pod's node was in   |
| `recently_started` | Boolean: was this pod started recently |

`recently_started` is controlled by the [environment variable](environment-variables.md) `RECENTLY_STARTED_POD_DURATION` and defaults to 10 seconds.

#### `pod_pending_count`

Total number of pods that started pending by reason.

|  attribute  |                 explanation                  |
|-------------|----------------------------------------------|
| `reason`    | Summary of the kubernetes Reason for pending |
| `namespace` | The namespace that the pod is in             |

#### `pod_restarts_total`

Total number of pods automatically restarted due to infrastructure failures before the main container started.
This counter tracks pods that were automatically restarted by the [failed pod restart](pod-restarts.md) feature.
These are infrastructure-level failures (like node eviction) that occur before the main container enters the Running state.

|       attribute        |                                                 explanation                                                 |
|------------------------|-------------------------------------------------------------------------------------------------------------|
| `reason`               | The infrastructure failure reason: `Evicted`, `NodeShutdown`, `NodeAffinity`, or `UnexpectedAdmissionError` |
| `condition` (optional) | The node condition that caused the pod restart, e.g., `DiskPressure`, `MemoryPressure`                      |
| `namespace`            | The namespace that the pod is in                                                                            |

`reason` will be one of:

- `Evicted`: Node pressure eviction (`DiskPressure`, `MemoryPressure`, etc.)
- `NodeShutdown`: Graceful node shutdown
- `NodeAffinity`: Node affinity/selector no longer matches
- `UnexpectedAdmissionError`: Unexpected error during pod admission

`condition` is extracted from the pod status message when available (e.g., `DiskPressure`, `MemoryPressure`).
It will be empty if the condition cannot be determined.

#### `pods_gauge`

A gauge of the number of workflow created pods currently in the cluster in each phase.
It is possible for a workflow to start, but no pods be running (for example cluster is too busy to run them).
This metric sheds light on actual work being done.

| attribute |         explanation          |
|-----------|------------------------------|
| `phase`   | The phase that the pod is in |

#### `pods_total_count`

Total number of pods that have entered each phase.

|  attribute  |           explanation            |
|-------------|----------------------------------|
| `phase`     | The phase that the pod is in     |
| `namespace` | The namespace that the pod is in |

This metric ignores the `PodInitializing` reason and does not count it.
The `reason` attribute is the value from the Reason message before the `:` in the message.
This is not directly controlled by the workflow controller, so it is possible for some pod pending states to be missed.

#### `queue_adds_count`

A counter of additions to the work queues inside the controller.
The rate of this shows how busy that area of the controller is

|  attribute   |      explanation      |
|--------------|-----------------------|
| `queue_name` | The name of the queue |

Queues:

- `cron_wf_queue`: the queue of CronWorkflow updates from the cluster
- `pod_cleanup_queue`: pods which are queued for deletion
- `workflow_queue`: the queue of Workflow updates from the cluster
- `workflow_ttl_queue`: workflows which are queued for deletion due to age
- `workflow_archive_queue`: workflows which are queued for archiving

This and associated metrics are all directly sourced from the [client-go workqueue metrics](https://godocs.io/k8s.io/client-go/util/workqueue)

#### `queue_depth_gauge`

A gauge of the current depth of the queues.
If these get large then the workflow controller is not keeping up with the cluster.

|  attribute   |      explanation      |
|--------------|-----------------------|
| `queue_name` | The name of the queue |

Queues:

- `cron_wf_queue`: the queue of CronWorkflow updates from the cluster
- `pod_cleanup_queue`: pods which are queued for deletion
- `workflow_queue`: the queue of Workflow updates from the cluster
- `workflow_ttl_queue`: workflows which are queued for deletion due to age
- `workflow_archive_queue`: workflows which are queued for archiving

This and associated metrics are all directly sourced from the [client-go workqueue metrics](https://godocs.io/k8s.io/client-go/util/workqueue)

#### `queue_duration`

A histogram of the time events in the queues are taking to be processed.

|  attribute   |      explanation      |
|--------------|-----------------------|
| `queue_name` | The name of the queue |

Default bucket sizes: 0.1, 0.2, 0.5, 1, 2, 5, 10, 20, 60, 180

Queues:

- `cron_wf_queue`: the queue of CronWorkflow updates from the cluster
- `pod_cleanup_queue`: pods which are queued for deletion
- `workflow_queue`: the queue of Workflow updates from the cluster
- `workflow_ttl_queue`: workflows which are queued for deletion due to age
- `workflow_archive_queue`: workflows which are queued for archiving

This and associated metrics are all directly sourced from the [client-go workqueue metrics](https://godocs.io/k8s.io/client-go/util/workqueue)

#### `queue_latency`

A histogram of the time events in the queues are taking before they are processed.

|  attribute   |      explanation      |
|--------------|-----------------------|
| `queue_name` | The name of the queue |

Default bucket sizes: 1, 5, 20, 60, 180

Queues:

- `cron_wf_queue`: the queue of CronWorkflow updates from the cluster
- `pod_cleanup_queue`: pods which are queued for deletion
- `workflow_queue`: the queue of Workflow updates from the cluster
- `workflow_ttl_queue`: workflows which are queued for deletion due to age
- `workflow_archive_queue`: workflows which are queued for archiving

This and associated metrics are all directly sourced from the [client-go workqueue metrics](https://godocs.io/k8s.io/client-go/util/workqueue)

#### `queue_longest_running`

A gauge of the number of seconds that this queue's longest running processor has been running for.

|  attribute   |      explanation      |
|--------------|-----------------------|
| `queue_name` | The name of the queue |

Queues:

- `cron_wf_queue`: the queue of CronWorkflow updates from the cluster
- `pod_cleanup_queue`: pods which are queued for deletion
- `workflow_queue`: the queue of Workflow updates from the cluster
- `workflow_ttl_queue`: workflows which are queued for deletion due to age
- `workflow_archive_queue`: workflows which are queued for archiving

This and associated metrics are all directly sourced from the [client-go workqueue metrics](https://godocs.io/k8s.io/client-go/util/workqueue)

#### `queue_retries`

A counter of the number of times a message has been retried in the queue.

|  attribute   |      explanation      |
|--------------|-----------------------|
| `queue_name` | The name of the queue |

Queues:

- `cron_wf_queue`: the queue of CronWorkflow updates from the cluster
- `pod_cleanup_queue`: pods which are queued for deletion
- `workflow_queue`: the queue of Workflow updates from the cluster
- `workflow_ttl_queue`: workflows which are queued for deletion due to age
- `workflow_archive_queue`: workflows which are queued for archiving

This and associated metrics are all directly sourced from the [client-go workqueue metrics](https://godocs.io/k8s.io/client-go/util/workqueue)

#### `queue_unfinished_work`

A gauge of the number of queue items that have not been processed yet.

|  attribute   |      explanation      |
|--------------|-----------------------|
| `queue_name` | The name of the queue |

Queues:

- `cron_wf_queue`: the queue of CronWorkflow updates from the cluster
- `pod_cleanup_queue`: pods which are queued for deletion
- `workflow_queue`: the queue of Workflow updates from the cluster
- `workflow_ttl_queue`: workflows which are queued for deletion due to age
- `workflow_archive_queue`: workflows which are queued for archiving

This and associated metrics are all directly sourced from the [client-go workqueue metrics](https://godocs.io/k8s.io/client-go/util/workqueue)

#### `resource_rate_limiter_latency`

A histogram of the delay duration from the resource creation rate limiter.
Records the delay that would be required before a resource creation operation could proceed.
This metric helps identify when the resource rate limiter is throttling pod creation.
A delay of 0 means the operation was allowed immediately. A non-zero delay indicates
the operation was rate limited.
This rate limiter is not on by default

This metric has no attributes.

Default bucket sizes: 0, 0.1, 0.5, 1, 5, 10, 30, 60, 180

#### `total_count`

A counter of workflows that have entered each phase for tracking them through their life-cycle, by namespace.

|  attribute  |               explanation               |
|-------------|-----------------------------------------|
| `phase`     | The phase that the Workflow has entered |
| `namespace` | The namespace that the Workflow is in   |

#### `version`

Build metadata for this Controller.

|    attribute     |                                              explanation                                              |
|------------------|-------------------------------------------------------------------------------------------------------|
| `version`        | The version of Argo                                                                                   |
| `platform`       | The [Go platform](https://go.dev/doc/install/source#environment) compiled for. Example: `linux/amd64` |
| `go_version`     | Version of Go used                                                                                    |
| `build_date`     | Build date                                                                                            |
| `compiler`       | The compiler used. Example: `gc`                                                                      |
| `git_commit`     | The full Git SHA1 commit                                                                              |
| `git_tree_state` | Whether the Git tree was `dirty` or `clean` when built                                                |
| `git_tag`        | The Git tag or `untagged` if it was not tagged                                                        |

#### `workers_busy_count`

A gauge of queue workers that are busy.

|   attribute   |    explanation    |
|---------------|-------------------|
| `worker_type` | The type of queue |

Worker Types:

- `cron_wf_queue`: the queue of CronWorkflow updates from the cluster
- `pod_cleanup_queue`: pods which are queued for deletion
- `workflow_queue`: the queue of Workflow updates from the cluster
- `workflow_ttl_queue`: workflows which are queued for deletion due to age
- `workflow_archive_queue`: workflows which are queued for archiving

This and associated metrics are all directly sourced from the [client-go workqueue metrics](https://godocs.io/k8s.io/client-go/util/workqueue)

#### `workflow_condition`

A gauge of the number of workflows with different conditions.
This will tell you the number of workflows with running pods.

| attribute |                    explanation                     |
|-----------|----------------------------------------------------|
| `type`    | The type of condition, currently only `PodRunning` |
| `status`  | Boolean: `true` or `false`                         |

#### `workflowtemplate_runtime`

A histogram of the runtime of workflows using `workflowTemplateRef` only.
Counts both WorkflowTemplate and ClusterWorkflowTemplate usage.
Records time between entering the `Running` phase and completion, so does not include any time in `Pending`.

|    attribute    |                         explanation                         |
|-----------------|-------------------------------------------------------------|
| `name`          | ⚠️ The name of the WorkflowTemplate/ClusterWorkflowTemplate. |
| `namespace`     | The namespace that the WorkflowTemplate is in               |
| `cluster_scope` | A boolean set true if this is a ClusterWorkflowTemplate     |

#### `workflowtemplate_triggered_total`

A counter of workflows using `workflowTemplateRef` only, as they enter each phase.
Counts both WorkflowTemplate and ClusterWorkflowTemplate usage.

|    attribute    |                         explanation                         |
|-----------------|-------------------------------------------------------------|
| `name`          | ⚠️ The name of the WorkflowTemplate/ClusterWorkflowTemplate. |
| `namespace`     | The namespace that the WorkflowTemplate is in               |
| `cluster_scope` | A boolean set true if this is a ClusterWorkflowTemplate     |
| `phase`         | The phase that the Workflow has entered                     |
<!-- Generated documentation END -->

### Metric types

Please see the [Prometheus docs on metric types](https://prometheus.io/docs/concepts/metric_types/).

### How metrics work in Argo

In order to analyze the behavior of a workflow over time, you need to be able to link different instances (individual executions) of a workflow together into a "series" for the purposes of emitting metrics.
You can do this by linking them together with the same metric descriptor.

In Prometheus, a metric descriptor is defined as a metric's name and its key-value labels.
For example, for a metric tracking the duration of model execution over time, a metric descriptor could be:

`argo_workflows_model_exec_time{model_name="model_a",phase="validation"}`

This metric then represents the amount of time that "Model A" took to train in the phase "Validation".
It is important to understand that the metric name _and_ its labels form the descriptor: `argo_workflows_model_exec_time{model_name="model_b",phase="validation"}`is a different metric (and will track a different "series" altogether).

Now, whenever you run a workflow that validates "Model A" a metric with the amount of time it took it to do so will be created and emitted.
For each subsequent time that this happens, no new metrics will be emitted and the _same_ metric will be updated with the new value.
Since, you are interested on the execution time of "validation" of "Model A" over time, you are  no longer interested in the previous metric and can assume it has already been stored.

In summary, whenever you want to track a particular metric over time, you should use the same metric name _and_ metric labels wherever it is emitted.
This is how these metrics are "linked" as belonging to the same series.

### Grafana Dashboard for Argo Controller Metrics

Please see the [Argo Workflows metrics](https://grafana.com/grafana/dashboards/21393-argo-workflows-metrics-3-6/) Grafana dashboard.
