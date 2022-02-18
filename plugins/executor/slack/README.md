<!-- This is an auto-generated file. DO NOT EDIT -->
# slack

* Needs: >= v3.3
* Image: python:alpine

This plugin sends a Slack message.

You must create a secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: slack-executor-plugin
stringData:
  URL: https://hooks.slack.com/services/.../.../...
```

Example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: slack-example-
spec:
  entrypoint: main
  templates:
    - name: main
      plugin:
        slack:
          text: "{{workflow.name}} finished!"
```


Install:

    kubectl apply -f slack-executor-plugin-configmap.yaml

Uninstall:
	
    kubectl delete cm slack-executor-plugin 
