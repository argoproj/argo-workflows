Description: Continue an existing OpenTelemetry trace from trace context annotations on the workflow
Authors: [Vilmos Nagy](https://github.com/vilmosnagy)
Component: Telemetry
Issues: 17117

Set the `opentelemetry.io/traceparent` annotation on a workflow, for example `opentelemetry.io/traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01`, to make the workflow part of that trace.
Use this when a traced application submits workflows, so its request, the workflow and the services the workflow's steps call show up as one trace.
The application can produce the annotation with its OpenTelemetry SDK's propagator, prefixing each key with `opentelemetry.io/`.
If the annotation is missing or malformed, the workflow starts a new trace, as before.
See [Tracing](https://argo-workflows.readthedocs.io/en/latest/tracing/#continuing-an-existing-trace) for details.
