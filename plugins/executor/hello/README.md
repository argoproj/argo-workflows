<!-- This is an auto-generated file. DO NOT EDIT -->
# hello

* Needs: >= v3.3
* Image: python:alpine

This is the "hello world" plugin.

Example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-example-
spec:
  entrypoint: main
  templates:
    - name: main
      plugin:
        hello: { }
```


Install:

    kubectl apply -f hello-executor-plugin-configmap.yaml

Uninstall:
	
    kubectl delete cm hello-executor-plugin 
