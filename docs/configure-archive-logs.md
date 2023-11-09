# Configuring Archive Logs

⚠️ We do not recommend you rely on Argo Workflows to archive logs. Instead, use a conventional Kubernetes logging facility.

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
