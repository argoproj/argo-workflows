# Prometheus Metrics

> v2.7 and after

## Introduction

Argo emits a certain number of controller metrics that inform on the state of the controller at any given time. Furthermore,
users can also define their own custom metrics to inform on the state of their Workflows.

Custom Prometheus metrics can be defined to be emitted on a `Workflow`- and `Template`-level basis. These can be useful
for many cases; some examples:

- Keeping track of the duration of a `Workflow` or `Template` over time, and setting an alert if it goes beyond a threshold
- Keeping track of the number of times a `Workflow` or `Template` fails over time
- Reporting an important internal metric, such as a model training score or an internal error rate

Emitting custom metrics with Argo is easy, but it's important to understand what makes a good Prometheus metric and the
best way to define metrics in Argo to avoid problems such as [cardinality explosion](https://stackoverflow.com/questions/46373442/how-dangerous-are-high-cardinality-labels-in-prometheus).

## Metrics and metrics in Argo

There are two kinds of metrics emitted by Argo: **controller metrics** and **custom metrics**.

### Controller metrics

Metrics that inform on the state of the controller; i.e., they answer the question "What is the state of the controller right now?"
Default controller metrics can be scraped from service ```workflow-controller-metrics``` at the endpoint ```<host>:9090/metrics```

### Custom metrics

Metrics that inform on the state of a Workflow, or a series of Workflows. These custom metrics are defined by the user in the Workflow spec.

Emitting custom metrics is the responsibility of the emitter owner. Since the user defines Workflows in Argo, the user is responsible
for emitting metrics correctly.

### What is and isn't a Prometheus metric

Prometheus metrics should be thought of as ephemeral data points of running processes; i.e., they are the answer to
the question "What is the state of my system _right now_?". Metrics should report things such as:

- a counter of the number of times a workflow or steps has failed, or
- a gauge of workflow duration, or
- an average of an internal metric such as a model training score or error rate.

Metrics are then routinely scraped and stored and -- when they are correctly designed -- they can represent time series.
Aggregating the examples above over time could answer useful questions such as:

- How has the error rate of this workflow or step changed over time?
- How has the duration of this workflow changed over time? Is the current workflow running for too long?
- Is our model improving over time?

Prometheus metrics should **not** be thought of as a store of data. Since metrics should only report the state of the system
at the current time, they should not be used to report historical data such as:

- the status of an individual instance of a workflow, or
- how long a particular instance of a step took to run.

Metrics are also ephemeral, meaning there is no guarantee that they will be persisted for any amount of time. If you need
a way to view and analyze historical data, consider the [workflow archive](workflow-archive.md) or reporting to logs.

### Default Controller Metrics

Metrics for the Four Golden Signals are:

- Latency: `argo_workflows_queue_latency`
- Traffic: `argo_workflows_count` and `argo_workflows_queue_depth_count`
- Errors: `argo_workflows_count` and `argo_workflows_error_count`
- Saturation: `argo_workflows_workers_busy` and `argo_workflows_workflow_condition`

<!-- titles should be the exact metric name for deep-linking, alphabetical ordered -->

#### `argo_pod_missing`

Pods were not seen. E.g. by being deleted by Kubernetes. You should only see this under high load.

!!! NOTE
    This metric's name starts with `argo_` not `argo_workflows_`.

#### `argo_workflows_count`

Number of workflow in each phase. The `Running` count does not mean that a workflows pods are running, just that the controller has scheduled them. A workflow can be stuck in `Running` with pending pods for a long time.

#### `argo_workflows_error_count`

A count of certain errors incurred by the controller.

#### `argo_workflows_k8s_request_total`

Number of API requests sent to the Kubernetes API.

#### `argo_workflows_operation_duration_seconds`

A histogram of durations of operations.

#### `argo_workflows_pods_count`

It is possible for a workflow to start, but no pods be running (e.g. cluster is too busy to run them). This metric sheds light on actual work being done.

#### `argo_workflows_queue_adds_count`

The number of additions to the queue of workflows or cron workflows.

#### `argo_workflows_queue_depth_count`

The depth of the queue of workflows or cron workflows to be processed by the controller.

#### `argo_workflows_queue_latency`

The time workflows or cron workflows spend in the queue waiting to be processed.

#### `argo_workflows_workers_busy`

The number of workers that are busy.

#### `argo_workflows_workflow_condition`

The number of workflow with different conditions. This will tell you the number of workflows with running pods.

#### `argo_workflows_workflows_processed_count`

A count of all Workflow updates processed by the controller.

### Metric types

Please see the [Prometheus docs on metric types](https://prometheus.io/docs/concepts/metric_types/).

### How metrics work in Argo

In order to analyze the behavior of a workflow over time, we need to be able to link different instances
(i.e. individual executions) of a workflow together into a "series" for the purposes of emitting metrics. We do so by linking them together
with the same metric descriptor.

In Prometheus, a metric descriptor is defined as a metric's name and its key-value labels. For example, for a metric
tracking the duration of model execution over time, a metric descriptor could be:

`argo_workflows_model_exec_time{model_name="model_a",phase="validation"}`

This metric then represents the amount of time that "Model A" took to train in the phase "Validation". It is important
to understand that the metric name _and_ its labels form the descriptor: `argo_workflows_model_exec_time{model_name="model_b",phase="validation"}`
is a different metric (and will track a different "series" altogether).

Now, whenever we run our first workflow that validates "Model A" a metric with the amount of time it took it to do so will
be created and emitted. For each subsequent time that this happens, no new metrics will be emitted and the _same_ metric
will be updated with the new value. Since, in effect, we are interested on the execution time of "validation" of "Model A"
over time, we are no longer interested in the previous metric and can assume it has already been scraped.

In summary, whenever you want to track a particular metric over time, you should use the same metric name _and_ metric
labels wherever it is emitted. This is how these metrics are "linked" as belonging to the same series.

### Grafana Dashboard for Argo Controller Metrics

Please see the [Argo Workflows metrics](https://grafana.com/grafana/dashboards/13927) Grafana dashboard.

## Defining metrics

Metrics are defined in-place on the Workflow/Step/Task where they are emitted from. Metrics are always processed _after_
the Workflow/Step/Task completes, with the exception of [real-time metrics](#real-time-metrics).

Metric definitions **must** include a `name` and a `help` doc string. They can also include any number of `labels` (when
defining labels avoid cardinality explosion). Metrics with the same `name` **must always** use the same exact `help` string,
having different metrics with the same name, but with a different `help` string will cause an error (this is a Prometheus requirement).

All metrics can also be conditionally emitted by defining a `when` clause. This `when` clause works the same as elsewhere
in a workflow.

A metric must also have a type, it can be one of `gauge`, `histogram`, and `counter` ([see below](#metric-spec)). Within
the metric type a `value` must be specified. This value can be either a literal value of be an [Argo variable](variables.md).

When defining a `histogram`, `buckets` must also be provided (see below).

[Argo variables](variables.md) can be included anywhere in the metric spec, such as in `labels`, `name`, `help`, `when`, etc.

Metric names can only contain alphanumeric characters, `_`, and `:`.

### Metric Spec

In Argo you can define a metric on the `Workflow` level or on the `Template` level. Here is an example of a `Workflow`
level Gauge metric that will report the Workflow duration time:

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

Argo supports a limited number of real-time metrics. These metrics are emitted in real-time, beginning when the step execution starts
and ending when it completes. Real-time metrics are only available on Gauge type metrics and with a [limited number of variables](variables.md#realtime-metrics).

To define a real-time metric simply add `realtime: true` to a gauge metric with a valid real-time variable. For example:

```yaml
  gauge:
    realtime: true
    value: "{{duration}}"
```

## Metrics endpoint

By default, metrics are emitted by the workflow-controller on port 9090 on the `/metrics` path. By port-forwarding to the pod you can view the metrics in your browser at `http://localhost:9090/metrics`:

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

## Metrics configuration

You can adjust various elements of the metrics configuration by changing values in the [Workflow Controller Config Map](workflow-controller-configmap.md).

```yaml
metricsConfig: |
  # Enabled controls metric emission. Default is true, set "enabled: false" to turn off
  enabled: true

  # Path is the path where metrics are emitted. Must start with a "/". Default is "/metrics"
  path: /metrics
  
  # Port is the port where metrics are emitted. Default is "9090"
  port: 8080

  # MetricsTTL sets how often custom metrics are cleared from memory. Default is "0", metrics are never cleared
  metricsTTL: "10m"

  # IgnoreErrors is a flag that instructs prometheus to ignore metric emission errors. Default is "false"
  ignoreErrors: false

  # Use a self-signed cert for TLS, default false
  secure: false
```
