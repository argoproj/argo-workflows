# Configuring Archive Logs

!!! Warning "Not recommended"
    We [do not recommend](#why-doesnt-argo-workflows-recommend-using-this-feature) you rely on Argo Workflows to archive logs. Instead, use a dedicated Kubernetes capable logging facility.

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
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
```

## Configuring Workflow Template

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: archive-location-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
    archiveLocation:
      archiveLogs: true
```

## Why doesn't Argo Workflows recommend using this feature?

Argo Workflows log storage facilities are quite basic. It is recommended that you use a combination of:

* A fully featured Kubernetes capable logging facility which will provide you with facilities for indexing, searching and managing of log storage.
* Use [links](links.md) to connect from the Argo Workflows user interface to your logging facility
    * Use the `scope: workflow` link to get all the logs for a workflow, using the workflow name in the link `${metadata.name}` and the namespace `${metadata.namespace}`
    * Use `scope: pod-logs` for those from a specific pod of name `${metadata.name}`

There is no intention to substantially improve the logging facilities provided by Argo Workflows, this is considered best implemented in a separate product.
