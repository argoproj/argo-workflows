# Configuring Archive Logs

To enable automatic pipeline logging, you need to configure ***archiveLogs*** at workflow-controller configmap, workflow spec, or template level. You also need to configure [Artifact Repository](configure-artifact-repository.md) to define where this logging artifact is stored.

Archive logs follows priorities:

workflow-controller config (on) > workflow spec (on/off) > template (on/off)

| Controller Configmap | Workflow Spec | Template | are we archiving logs? |
|---|---|---|---|
| true | true | true | true |
| true | true | false | true |
| true | false | true | true |
| true | false | false | true |
| false | true | true | true |
| false | true | false | false |
| false | false | true | true |
| false | false | false | false |

## Configuring Workflow Controller Configmap

See [Workflow Controller Configmap](workflow-controller-configmap.md)

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
