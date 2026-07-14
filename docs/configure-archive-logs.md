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

## Archiving Init and Wait Container Logs

By default, only the `main` container's logs are archived.
In the legacy pod layout, every Workflow Pod also runs Argo's own system containers — an `init` container (which loads input artifacts) and a `wait` container (which saves outputs and logs) — whose logs are normally not archived.

Set `archiveSystemContainerLogs: true` to also archive the `init` and `wait` container logs.
They are stored alongside `main-logs` as artifacts named `init-logs` and `wait-logs`, and can be viewed from the Argo UI for garbage-collected Pods.
This is mainly useful for inspecting what the system containers did after the Pod is gone, such as debugging artifact/output upload failures in `wait`, or reviewing which input artifacts the `init` container loaded.
Note that a *failing* `init` container is not captured (see [Limitations](#limitations) below).

`archiveSystemContainerLogs` is controlled separately from `archiveLogs` so that you only pay to store these extra logs when you need them.
It can be set at the workflow-controller config-map, workflow spec, or template level, and follows the same priorities as `archiveLogs`:

workflow-controller config (on) > workflow spec (on/off) > template (on/off)

The two settings are independent: you can archive the system container logs without archiving the `main` container logs (`archiveLogs: false`, `archiveSystemContainerLogs: true`), and vice versa.

In [init-less pod mode](initless-pod.md) there is no separate `init` and `wait` container — a single `supervisor` container performs both roles.
In that mode `archiveSystemContainerLogs: true` archives the supervisor's log as a single artifact named `supervisor-logs` instead of `init-logs` and `wait-logs`.

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
      archiveSystemContainerLogs: true
```

### Limitations

* **`init` failures are not archived.**
  If the `init` container fails, the `wait` container never starts, so the archiving step does not run and `init-logs` is not produced.
  Kubernetes runs init containers sequentially before the main containers, so this cannot be worked around.
* **The `wait` log is a best-effort snapshot.**
  The `wait` container archives its own log, so the few final lines produced while it is uploading its own log (and any work that happens after, such as reporting outputs) are not included.
* **Abrupt `wait` termination is not archived.**
  If the `wait` container is killed (for example `OOMKilled` or node eviction), the archiving step does not run.

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
