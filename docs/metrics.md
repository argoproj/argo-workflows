# Prometheus Metrics

> v2.7 and after

!!! Metrics changes in 3.6
    Please read [this short guide](upgrading.md#metrics-changes) on what you must consider when upgrading to 3.6.

## Introduction

Argo emits a certain number of controller metrics that inform on the state of the controller at any given time.
Furthermore, users can also define their own custom metrics to inform on the state of their Workflows.

Custom metrics can be defined to be emitted on a `Workflow`- and `Template`-level basis. These can be useful for many cases; some examples:

- Keeping track of the duration of a `Workflow` or `Template` over time, and setting an alert if it goes beyond a threshold
- Keeping track of the number of times a `Workflow` or `Template` fails over time
- Reporting an important internal metric, such as a model training score or an internal error rate

Emitting custom metrics with Argo is easy, but it's important to understand what makes a good metric and the best way to define metrics in Argo to avoid problems such as [cardinality explosion](https://stackoverflow.com/questions/46373442/how-dangerous-are-high-cardinality-labels-in-prometheus).

Metrics can be collected using the OpenTelemetry protocol or via Prometheus compatible scraping.

## Metrics configuration

It is possible to collect metrics via the OpenTelemetry protocol or via Prometheus compatible scraping.
Both of these mechanisms can be enabled at the same time, which could be useful if you'd like to migrate from one system to the other.
Using multiple protocols at the same time is not intended for long term use.

OpenTelemetry is the recommended way of collecting metrics.
The OpenTelemetry collector can export metrics to Prometheus via [the Prometheus remote write exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/prometheusremotewriteexporter/README.md).

### OpenTelemetry protocol

To enable the OpenTelemetry protocol you must set the environment variable `OTEL_EXPORTER_OTLP_ENDPOINT` or `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT`.
It will not be enabled if left blank, unlike some other implementations.

You can configure the protocol using the environment variables documented in [standard environment variables](https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/).

The [configuration options](#common) in the controller ConfigMap `metricsTTL`, `modifiers` and `temporality` affect the OpenTelemetry behavior, but the other parameters do not.

To use the [OpenTelemetry collector](https://opentelemetry.io/docs/collector/) you can configure it

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
```

You can use the [OpenTelemetry operator](https://opentelemetry.io/docs/kubernetes/operator/) to setup the collector and instrument the workflow-controller.

You can configure the [temporality](https://opentelemetry.io/docs/specs/otel/metrics/data-model/#temporality) of OpenTelemetry metrics in the [Workflow Controller ConfigMap](workflow-controller-configmap.md).

```yaml
metricsConfig: |
  # >= 3.6. Which temporality to use for OpenTelemetry. Default is "Cumulative"
  temporality: Delta
```

### Prometheus scraping

You can adjust various elements of the Prometheus metrics configuration by changing values in the [Workflow Controller Config Map](workflow-controller-configmap.md).

```yaml
metricsConfig: |
  # Enabled controls the prometheus metric. Default is true, set "enabled: false" to turn off
  enabled: true

  # Path is the path where prometheus metrics are emitted. Must start with a "/". Default is "/metrics"
  path: /metrics

  # Port is the port where prometheus metrics are emitted. Default is "9090"
  port: 8080

  # IgnoreErrors is a flag that instructs prometheus to ignore metric emission errors. Default is "false"
  ignoreErrors: false

  # Use a self-signed cert for TLS
  # >= 3.6: default true
  secure: true
```

The metric names emitted by this mechanism are prefixed with `argo_workflows_`.
`Attributes` are exposed as Prometheus `labels` of the same name.

Prometheus metrics will return empty metrics on a workflow controller which is not the leader.
All metrics emitted over Prometheus will have `argo_workflows_` prefixed to their name.

### Common

You can adjust various elements of the metrics configuration by changing values in the [Workflow Controller Config Map](workflow-controller-configmap.md).

```yaml
metricsConfig: |
  # MetricsTTL sets how often custom metrics are cleared from memory. Default is "0", metrics are never cleared. Histogram metrics are never cleared.
  metricsTTL: "10m"
  # Modifiers allows tuning of each of the emitted metrics
  modifiers:
    pod_missing:
      disabled: true
    cronworkflows_triggered_total:
      disabledAttributes:
        - name
    k8s_request_duration:
      histogramBuckets: [ 1.0, 2.0, 10.0 ]
```

### Modifiers

Using modifiers you can manipulate the metrics created by the workflow controller.
These modifiers apply to the built-in metrics and any custom metrics you create.
Each modifier applies to the named metric only, and to all output methods.

`disabled: true` will disable the emission of the metric from the system.

```yaml
  disabledAttributes:
    - namespace
```

Will disable the attribute (label) from being emitted.
The metric will be emitted with the attribute missing, the remaining attributes will still be emitted with the values correctly aggregated.
This can be used to reduce cardinality of metrics.

```yaml
  histogramBuckets:
    - 1.0
    - 2.0
    - 5.0
    - 10.0
```

For histogram metrics only, this will change the boundary values for the histogram buckets.
All values must be floating point numbers.

## Metrics and metrics in Argo

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

!!! High cardinality
    Some metric attributes may have high cardinality and are marked with ⚠️ to warn you. You may need to disable this metric or disable the attribute.
<!-- titles should be the exact metric name for deep-linking, alphabetical ordered -->
<!-- titles are without argo_workflows prefix -->
#### `cronworkflows_triggered_total`

A counter of the number of times a CronWorkflow has been

| attribute   | explanation                               |
|-------------|-------------------------------------------|
| `name`     | ⚠️ The name of the CronWorkflow. |
| `namespace` | The namespace in which the pod is running |

#### `gauge`

A gauge of the number of workflows currently in the cluster in each phase. The `Running` count does not mean that a workflows pods are running, just that the controller has scheduled them. A workflow can be stuck in `Running` with pending pods for a long time.

| attribute | explanation                       |
|-----------|-----------------------------------|
| `status`  | The phase that the workflow is in |

#### `error_count`

A counter of certain errors incurred by the controller.

| attribute | explanation            |
|-----------|------------------------|
| `cause`     | The cause of the error |

The currently tracked specific errors are

- `OperationPanic` - the controller `panic()` on a programming bug
- `CronWorkflowSubmissionError` - A cron workflow failed submission
- `CronWorkflowSpecError` - A cron workflow has an invalid specification

#### `k8s_request_total`

A counter of the number of API requests sent to the Kubernetes API.

| attribute     | explanation                                                        |
|---------------|--------------------------------------------------------------------|
| `kind`        | The kubernetes `kind` involved in the request such as `configmaps` |
| `verb`        | The verb of the request, such as `Get` or `List`                   |
| `status_code` | The HTTP status code of the response                               |

This metric is calculable from `k8s_request_duration`, and it is suggested you just collect that metric instead.

#### `k8s_request_duration`

A histogram recording how long each type of request took.

| attribute     | explanation                                                        |
|---------------|--------------------------------------------------------------------|
| `kind`        | The kubernetes `kind` involved in the request such as `configmaps` |
| `verb`        | The verb of the request, such as `Get` or `List`                   |
| `status_code` | The HTTP status code of the response                               |

This is contains all the information contained in `k8s_request_total` along with timings.

#### `is_leader`

A gauge indicating if this Controller is the [leader](high-availability.md#workflow-controller).

- `1` if leader or in standalone mode via [`LEADER_ELECTION_DISABLE=true`](environment-variables.md#controller).
- `0` otherwise, indicating that this controller is a standby that is not currently running workflows.

#### `log_messages`

A count of log messages emitted by the controller by log level: `error`, `warn` and `info`.

| attribute | explanation                  |
|-----------|------------------------------|
| `level`   | The log level of the message |

#### `operation_duration_seconds`

A histogram of durations of operations. An operation is a single workflow reconciliation loop within the workflow-controller.
It's the time for the controller to process a single workflow after it has been read from the cluster and is a measure of the performance of the controller affected by the complexity of the workflow.

This metric has no attributes.

The environment variables `OPERATION_DURATION_METRIC_BUCKET_COUNT` and `MAX_OPERATION_TIME` configure the bucket sizes for this metric, unless they are specified using an `histogramBuckets` modifier in the `metricsConfig` block.

#### `pods_gauge`

A gauge of the number of workflow created pods currently in the cluster in each phase.
It is possible for a workflow to start, but no pods be running (for example cluster is too busy to run them).
This metric sheds light on actual work being done.

| attribute | explanation                  |
|-----------|------------------------------|
| `phase`   | The phase that the pod is in |

#### `pod_missing`

A counter of pods that were not seen - for example they are by being deleted by Kubernetes.
You should only see this under high load.

| attribute          | explanation                            |
|--------------------|----------------------------------------|
| `recently_started` | Boolean: was this pod started recently |
| `node_phase`       | The phase that the pod's node was in   |

`recently_started` is controlled by the [environment variable](environment-variables.md) `RECENTLY_STARTED_POD_DURATION` and defaults to 10 seconds.

#### `pod_pending_count`

A counter of pods that have been seen in the Pending state.

| attribute          | explanation                               |
|--------------------|-------------------------------------------|
| `reason` | Summary of the kubernetes Reason for pending.    |
| `namespace`        | The namespace in which the pod is running |

This metric ignores the `PodInitializing` reason and does not count it.
The `reason` attribute is the value from the Reason message before the `:` in the message.
This is not directly controlled by the workflow controller, so it is possible for some pod pending states to be missed.

#### `pods_total_count`

A gauge of the number of pods which have entered each phase and then observed by the controller.
This is not directly controlled by the workflow controller, so it is possible for some pod phases to be missed.

| attribute   | explanation                               |
|-------------|-------------------------------------------|
| `phase`     | The phase that the pod is in              |
| `namespace` | The namespace in which the pod is running |

#### `queue_adds_count`

A counter of additions to the work queues inside the controller.
The rate of this shows how busy that area of the controller is.

| attribute     | explanation       |
|---------------|-------------------|
| `worker_type` | The type of queue |

Queues:

- `workflow_queue`: the queue of Workflow updates from the cluster
- `cron_wf_queue`: the queue of CronWorkflow updates from the cluster
- `workflow_ttl_queue`: workflows which are queued for deletion due to age
- `pod_cleanup_queue`: pods which are queued for deletion

This and associated metrics are all directly sourced from the [client-go workqueue metrics](https://godocs.io/k8s.io/client-go/util/workqueue)

#### `queue_depth_gauge`

A gauge of the current depth of the queues.
If these get large then the workflow controller is not keeping up with the cluster.

See [queue adds count](#queue_adds_count) for details.

#### `queue_duration`

A histogram of the time events in the queues are taking to be processed.

See [queue adds count](#queue_adds_count) for details.

#### `queue_latency`

A histogram of the time events in the queues are taking before they are processed.

See [queue adds count](#queue_adds_count) for details.

#### `queue_longest_running`

A gauge of the number of seconds that this queue's longest running processor has been running for.

See [queue adds count](#queue_adds_count) for details.

#### `queue_retries`

A counter of the number of times a message has been retried in the queue

See [queue adds count](#queue_adds_count) for details.

#### `queue_unfinished_work`

A gauge of the number of queue items that have not been processed yet.

See [queue adds count](#queue_adds_count) for details.

#### `total_count`

A counter of workflows that have entered each phase for tracking them through their life-cycle, by namespace.

| attribute   | explanation                                    |
|-------------|------------------------------------------------|
| `phase`     | The phase that the workflow has entered        |
| `namespace` | The namespace in which the workflow is running |

#### `version`

Build metadata for this Controller.

| attribute        | explanation                                                                                           |
|------------------|-------------------------------------------------------------------------------------------------------|
| `version`        | The version of Argo                                                                                   |
| `platform`       | The [Go platform](https://go.dev/doc/install/source#environment) compiled for. Example: `linux/amd64` |
| `go_version`     | Version of Go used                                                                                    |
| `build_date`     | Build date                                                               |
| `compiler`       | The compiler used. Example: `gc`                                         |
| `git_commit`     | The full Git SHA1 commit                                                                              |
| `git_tree_state` | Whether the Git tree was `dirty` or `clean` when built                                                |
| `git_tag`        | The Git tag or `untagged` if it was not tagged                                                        |

#### `workers_busy_count`

A count of queue workers that are busy.

See [queue adds count](#queue_adds_count) for details.

#### `workflow_condition`

A gauge of the number of workflows with different conditions.
This will tell you the number of workflows with running pods.

| attribute | explanation                                     |
|-----------|-------------------------------------------------|
| `type`    | the type of condition, currently only `Running` |
| `status`  | `true` or `false`                               |

#### `workflowtemplate_runtime`

A histogram of the duration of workflows using `workflowTemplateRef` only, as they enter each phase.
Counts both WorkflowTemplate and ClusterWorkflowTemplate usage.
Records time between entering the `Running` phase and completion, so does not include any time in `Pending`.

| attribute       | explanation                                                  |
|-----------------|--------------------------------------------------------------|
| `cluster_scope` | A boolean set true if this is a ClusterWorkflowTemplate      |
| `name`          | ⚠️ The name of the WorkflowTemplate/ClusterWorkflowTemplate.  |
| `namespace`     | The namespace from which the WorkflowTemplate is being used  |

#### `workflowtemplate_triggered_total`

A counter of workflows using `workflowTemplateRef` only, as they enter each phase.
Counts both WorkflowTemplate and ClusterWorkflowTemplate usage.

| attribute       | explanation                                                  |
|-----------------|--------------------------------------------------------------|
| `cluster_scope` | A boolean set true if this is a ClusterWorkflowTemplate      |
| `name`          | ⚠️ The name of the WorkflowTemplate/ClusterWorkflowTemplate.  |
| `namespace`     | The namespace from which the WorkflowTemplate is being used  |
| `phase`         | The phase that the workflow entered                          |

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

Please see the [Argo Workflows metrics](https://grafana.com/grafana/dashboards/20348-argo-workflows-metrics/) Grafana dashboard.

## Defining custom metrics

Metrics are defined in-place on the Workflow/Step/Task where they are emitted from.
Metrics are always processed _after_ the Workflow/Step/Task completes, with the exception of [real-time metrics](#real-time-metrics).
Custom metrics are defined under a `prometheus` tag in the yaml for legacy reasons.
They are emitted over all active protocols.

Metric definitions **must** include a `name` and a `help` doc string.
They can also include any number of `labels` (when defining labels avoid cardinality explosion).
Metrics with the same `name` **must always** use the same exact `help` string, having different metrics with the same name, but with a different `help` string will cause an error (this is a Prometheus requirement).
Metrics with the same `name` may not change what type of metric they are.

All metrics can also be conditionally emitted by defining a `when` clause.
This `when` clause works the same as elsewhere in a workflow.

A metric must also have a type, it can be one of `gauge`, `histogram`, and `counter` ([see below](#metric-spec)).
Within the metric type a `value` must be specified. This value can be either a literal value of be an [Argo variable](variables.md).

When defining a `histogram`, `buckets` must also be provided (see below).

[Argo variables](variables.md) can be included anywhere in the metric spec, such as in `labels`, `name`, `help`, `when`, etc.

Metric names can only contain alphanumeric characters and `_` for compatibility with both Prometheus and OpenTelemetry, even if only one of these protocols is in use.

### Metric Spec

In Argo you can define a metric on the `Workflow` level or on the `Template` level.
Here is an example of a `Workflow` level Gauge metric that will report the Workflow duration time:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: model-training-
spec:
  entrypoint: steps
  metrics:
    prometheus:
      - name: exec_duration_gauge         # Metric name (will be prepended with "argo_workflows_")
        labels:                           # Labels are optional. Avoid cardinality explosion.
          - key: name
            value: model_a
        help: "Duration gauge by name"    # A help doc describing your metric. This is required.
        gauge:                            # The metric type. Available are "gauge", "histogram", and "counter".
          value: "{{workflow.duration}}"  # The value of your metric. It could be an Argo variable (see variables doc) or a literal value

...
```

Gauges take an optional `operation` flag which must be one of `Set`, `Add` or `Sub`. If this is unspecified it as though you have used `operation: Set`.

- `Set`: makes the gauge report the `value`
- `Add`: increases the current value of the gauge by `value`
- `Sub`: decreases the current value of the gauge by `value`

An example of a `Template`-level Counter metric that will increase a counter every time the step fails:

```yaml
...
  templates:
    - name: flakey
      metrics:
        prometheus:
          - name: result_counter
            help: "Count of step execution by result status"
            labels:
              - key: name
                value: flakey
            when: "{{status}} == Failed"       # Emit the metric conditionally. Works the same as normal "when"
            counter:
              value: "1"                            # This increments the counter by 1
      container:
        image: python:alpine3.6
        command: ["python", -c]
        # fail with a 66% probability
        args: ["import random; import sys; exit_code = random.choice([0, 1, 1]); sys.exit(exit_code)"]
...
```

The counter `value` is added to the previous value of the counter.

A similar example of such a Counter metric that will increase for every step status

```yaml
...
  templates:
    - name: flakey
      metrics:
        prometheus:
          - name: result_counter
            help: "Count of step execution by result status"
            labels:
              - key: name
                value: flakey
              - key: status
                value: "{{status}}"    # Argo variable in `labels`
            counter:
              value: "1"
      container:
        image: python:alpine3.6
        command: ["python", -c]
        # fail with a 66% probability
        args: ["import random; import sys; exit_code = random.choice([0, 1, 1]); sys.exit(exit_code)"]
...
```

Finally, an example of a `Template`-level Histogram metric that tracks an internal value:

```yaml
...
  templates:
    - name: random-int
      metrics:
        prometheus:
          - name: random_int_step_histogram
            help: "Value of the int emitted by random-int at step level"
            when: "{{status}} == Succeeded"    # Only emit metric when step succeeds
            histogram:
              buckets:                              # Bins must be defined for histogram metrics
                - 2.01                              # and are part of the metric descriptor.
                - 4.01                              # All metrics in this series MUST have the
                - 6.01                              # same buckets.
                - 8.01
                - 10.01
              value: "{{outputs.parameters.rand-int-value}}"         # References itself for its output (see variables doc)
      outputs:
        parameters:
          - name: rand-int-value
            globalName: rand-int-value
            valueFrom:
              path: /tmp/rand_int.txt
      container:
        image: alpine:latest
        command: [sh, -c]
        args: ["RAND_INT=$((1 + RANDOM % 10)); echo $RAND_INT; echo $RAND_INT > /tmp/rand_int.txt"]
...
```

### Real-Time Metrics

Argo supports a limited number of real-time metrics.
These metrics are emitted in real-time, beginning when the step execution starts and ending when it completes.
Real-time metrics are only available on Gauge type metrics and with a [limited number of variables](variables.md#real-time-metrics).

To define a real-time metric simply add `realtime: true` to a gauge metric with a valid real-time variable. For example:

```yaml
  gauge:
    realtime: true
    value: "{{duration}}"
```

## Metrics endpoint

By default, metrics are emitted by the workflow-controller on port 9090 on the `/metrics` path.
By port-forwarding to the pod you can view the metrics in your browser at `http://localhost:9090/metrics`:

`kubectl -n argo port-forward deploy/workflow-controller 9090:9090`

A metrics service is not installed as part of [the default installation](quick-start.md) so you will need to add one if you wish to use a Prometheus Service Monitor:

```yaml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  labels:
    app: workflow-controller
  name: workflow-controller-metrics
  namespace: argo
spec:
  ports:
  - name: metrics
    port: 9090
    protocol: TCP
    targetPort: 9090
  selector:
    app: workflow-controller
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: argo-workflows
  namespace: argo
spec:
  endpoints:
  - port: metrics
  selector:
    matchLabels:
      app: workflow-controller
EOF
```

If you have more than one controller pod, using one as a [hot-standby](high-availability.md), you should use [a headless service](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services) to ensure that each pod is being scraped so that no metrics are missed.
