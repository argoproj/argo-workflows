<!-- This is an auto-generated file. DO NOT EDIT -->
# python

* Needs: >= v3.3
* Image: python:alpine

This plugin runs trusted Python expressions.

Example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: python-example-
spec:
  entrypoint: main
  templates:
    - name: main
      plugin:
        python:
          expression: "{{workflow.name}} finished!"
```


Install:

    kubectl apply -f python-executor-plugin-configmap.yaml

Uninstall:
	
    kubectl delete cm python-executor-plugin 
