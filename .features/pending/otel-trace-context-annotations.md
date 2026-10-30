Description: Continue an existing OpenTelemetry trace from trace context annotations on the workflow
Authors: [Vilmos Nagy](https://github.com/vilmosnagy)
Component: Telemetry
Issues: 17117

Set the `opentelemetry.io/traceparent` annotation on a workflow, for example `opentelemetry.io/traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01`, to make the workflow part of that trace.
Use this when a traced application submits workflows, so its request and the workflow show up as one trace.
The application can produce the annotation with its OpenTelemetry SDK's propagator, prefixing each key with `opentelemetry.io/`.
The code a step runs can join the trace too, and pass it on to the services it calls, through the `TRACEPARENT` environment variable that every workflow pod gets.
If the annotation is missing or malformed, the workflow starts a new trace.
See [Tracing](https://argo-workflows.readthedocs.io/en/latest/tracing/#continuing-an-existing-trace) for details.
