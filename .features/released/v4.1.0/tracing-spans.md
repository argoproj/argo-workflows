Description: Workflow Tracing
Authors: [Alan Clucas](https://github.com/Joibel)
Component: General
Issues: 12077

Argo Workflows can now emit OpenTelemetry traces, letting you see exactly what's happening inside a workflow run -- from controller reconciliation down to individual artifact uploads and log saves. Traces follow execution across the controller and executor processes, so you get a single span tree covering DAG node scheduling, pod creation, synchronization locks, script capture, and everything in between. If your workloads also emit OTel traces, they'll show up nested in the right place. Configure the tracing section in your workflow-controller-configmap with a collector URL and point your Jaeger or Tempo instance at it.
