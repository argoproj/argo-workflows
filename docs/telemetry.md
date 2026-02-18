# Telemetry

For a hands-on walkthrough, see the [Getting Started with OpenTelemetry](telemetry-getting-started.md) guide.

Argo Workflows emits the standard signals: traces, metrics and logs.

## Metrics

The workflow-controller emits controller and workflow level metrics.
Metrics can be collected via OpenTelemetry protocol or Prometheus scraping.
How to configure metrics is documented [here](telemetry-configuration.md#metrics).

See [the metrics page](metrics.md) for details on available metrics and custom metrics.

The argo-server does provide metrics via Prometheus scraping, but this is otherwise completely undocumented.

## Tracing

Tracing can be configured for emitting spans showing how workflows perform.
The traces are transmitted from the workflow-controller and argoexec using the OpenTelemetry protocol.
How to configure tracing is documented [here](telemetry-configuration.md#tracing).

See [the tracing page](tracing.md) for details on the available spans.

There is no tracing support in argo-server yet.

## Logs

Logs are emitted from stderr from all components: argo-server, CLI, workflow-controller and argoexec.
How to configure logging is documented [here](telemetry-configuration.md#logs).
