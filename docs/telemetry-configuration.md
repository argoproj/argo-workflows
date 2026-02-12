# Telemetry

Argo Workflows emits the standard signals: traces, metrics and logs.

## Configuration

Both metrics and tracing use the OpenTelemetry protocol and can be configured through environment variables and the workflow-controller ConfigMap.
Logging is emitted via stderr.
Metrics can also be scraped using Prometheus.

### OpenTelemetry

This is common configuration for metrics and tracing.

To enable the OpenTelemetry protocol you must set the environment variable `OTEL_EXPORTER_OTLP_ENDPOINT`, or the signal-specific endpoints `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` and `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`.
It will not be enabled if left blank, unlike some other implementations.

You can configure the protocol using the environment variables documented in the [OpenTelemetry standard environment variables](https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/).

By default, GRPC is used; to switch to HTTP, set either the `OTEL_EXPORTER_OTLP_PROTOCOL` or signal-specific `OTEL_EXPORTER_OTLP_METRICS_PROTOCOL` / `OTEL_EXPORTER_OTLP_TRACES_PROTOCOL` environment variable to `http/protobuf`.

To use the [OpenTelemetry collector](https://opentelemetry.io/docs/collector/) you can configure it:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
```

You can use the [OpenTelemetry operator](https://opentelemetry.io/docs/kubernetes/operator/) to setup the collector and instrument the workflow-controller.
The OpenTelemetry operator can also instrument your workload pods so that they emit spans as part of workflow tracing and this is the recommended setup.

## Metrics

### OpenTelemetry Metrics

OpenTelemetry is the recommended way of collecting metrics.
The OpenTelemetry collector can also export metrics to Prometheus via [the Prometheus remote write exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/prometheusremotewriteexporter/README.md).
To configure OpenTelemetry see [the common configuration information](#opentelemetry).

You can configure the [temporality](https://opentelemetry.io/docs/specs/otel/metrics/data-model/#temporality) of OpenTelemetry metrics in the [Workflow Controller ConfigMap](workflow-controller-configmap.md):

```yaml
metricsConfig: |
  # >= 3.6. Which temporality to use for OpenTelemetry. Default is "Cumulative"
  temporality: Delta
```

The [configuration options](#common-metrics-settings) `metricsTTL`, `modifiers` and `temporality` affect the OpenTelemetry behavior, but the other Prometheus-specific parameters do not.

### Prometheus Scraping

A metrics service is not installed as part of [the default installation](quick-start.md) so you will need to add one if you wish to use a Prometheus Service Monitor.
If you have more than one controller pod, using one as a [hot-standby](high-availability.md), you should use [a headless service](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services) to ensure that each pod is being scraped so that no metrics are missed.

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
  clusterIP: None
  ports:
  - name: metrics
    port: 9090
    protocol: TCP
    targetPort: 9090
  selector:
z    app: workflow-controller
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

You can adjust various elements of the Prometheus metrics configuration by changing values in the [Workflow Controller Config Map](workflow-controller-configmap.md):

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

By port-forwarding to the leader controller Pod you can view the metrics in your browser at `https://localhost:9090/metrics`.
Assuming you only have one controller replica, you can port-forward with:

```bash
kubectl -n argo port-forward deploy/workflow-controller 9090:9090
```

!!! Note "UTF-8 in Prometheus metrics"
    Version `v3.7` upgraded the `github.com/prometheus/client_golang` library, changing the `NameValidationScheme` to `UTF8Validation`. This allows metric names to retain their original delimiters (e.g., .), instead of replacing them with underscores. To maintain the legacy behavior, you can set the environment variable `PROMETHEUS_LEGACY_NAME_VALIDATION_SCHEME`. For more details, refer to the official [Prometheus documentation](https://prometheus.io/docs/guides/utf8/).

### Common Metrics Settings

You can adjust various elements of the metrics configuration by changing values in the [Workflow Controller Config Map](workflow-controller-configmap.md):

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

#### Modifiers

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

## Tracing

Tracing is configured via OpenTelemetry environment variables.
To configure OpenTelemetry see [the common configuration information](#opentelemetry).

## Logs

Logs are emitted from stderr from all components: argo-server, CLI, workflow-controller and argoexec.
The log level is controlled by the following command line arguments:

| flag           | default | valid values                     |
|----------------|---------|----------------------------------|
| `--log-level`  | `info`  | `debug`, `info`, `warn`, `error` |
| `--log-format` | `text`  | `text`, `json`                   |
| `--gloglevel`  | `0`     | integer (e.g. `6`)               |

In general we'd recommend setting log-format to `json` for structured logging that's easier to query.

The workflow-controller passes all of these flags through to argoexec containers automatically.
See [the workflow controller ConfigMap](workflow-controller-configmap.md) for how to override them on argoexec (the executor).

`--gloglevel` sets the verbosity level of the Kubernetes client `klog` logger, which is separate from the Argo log level.
This controls the logging output from the Kubernetes Go client library.
You only need to set this for debugging Kubernetes API interactions.
