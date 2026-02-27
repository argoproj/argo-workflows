# Workflow telemetry

This page describes how workflow users and authors can use telemetry to understand their workflows.

## Metrics

A number of system level metrics

## Custom metrics

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
        image: python:alpine3.23
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
        image: python:alpine3.23
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
        image: alpine:3.23
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
