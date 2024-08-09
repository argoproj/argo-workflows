# Configuring Archive Logs

!!! Warning "Not recommended"
    We do not recommend relying on Argo to archive logs as it is naive and not purpose-built for indexing, searching, and storing logs.
    This feature is provided as a convenience to quickly view logs of garbage collected Pods in the Argo UI, but we [recommend](#suggested-alternatives) you integrate a dedicated, Kubernetes-aware logging facility.

To enable automatic pipeline logging, you need to configure `archiveLogs` at workflow-controller config-map, workflow spec, or template level. You also need to configure [Artifact Repository](configure-artifact-repository.md) to define where this logging artifact is stored.

Archive logs follows priorities:

workflow-controller config (on) > workflow spec (on/off) > template (on/off)

| Controller Config Map | Workflow Spec | Template | are we archiving logs? |
|-----------------------|---------------|----------|------------------------|
| true                  | true          | true     | true                   |
| true                  | true          | false    | true                   |
| true                  | false         | true     | true                   |
| true                  | false         | false    | true                   |
| false                 | true          | true     | true                   |
| false                 | true          | false    | false                  |
| false                 | false         | true     | true                   |
| false                 | false         | false    | false                  |

## Configuring Workflow Controller Config Map

See [Workflow Controller Config Map](workflow-controller-configmap.md)

## Configuring Workflow Spec

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: archive-location-
spec:
  archiveLogs: true
  entrypoint: hello-world
  templates:
  - name: hello-world
    container:
      image: busybox
      command: [echo]
      args: ["hello world"]
```

## Configuring Workflow Template

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: archive-location-
spec:
  entrypoint: hello-world
  templates:
  - name: hello-world
    container:
      image: busybox
      command: [echo]
      args: ["hello world"]
    archiveLocation:
      archiveLogs: true
```

## Suggested alternatives

Argo's log storage is naive and will not reach feature parity with purpose-built facilities optimized for indexing, searching, and storing logs. Some open-source tools include:

* [`fluentd`](https://github.com/fluent/fluentd) for collection
* [ELK](https://www.elastic.co/elastic-stack/) as storage, querying and a UI
* [`promtail`](https://grafana.com/docs/loki/latest/send-data/promtail/) for collection
* [`loki`](https://grafana.com/docs/loki/latest/) for storage and querying
* [`grafana`](https://grafana.com/docs/grafana/latest/) for a UI

You can add [links](links.md) to connect from the Argo UI to your logging facility's UI. See examples in the [`workflow-controller-configmap.yaml`](workflow-controller-configmap.yaml).

* Link `scope: workflow` to the logs of a Workflow
* Link `scope: pod-logs` to the logs of a specific Pod of a Workflow
* Parametrize the link with `${metadata.name}`, `${metadata.namespace}`, `${metadata.labels}`, and other available metadata
