# Workflow Templates

Workflow templates are reusable checks of YAML that you can use within your workflows. This allows you to have a library of templates.

You can create some example templates as follows:

```
argo template create https://raw.githubusercontent.com/argoproj/argo/master/examples/workflow-template/templates.yaml
```

The submit a workflow using one of those templates:

```
argo submit https://raw.githubusercontent.com/argoproj/argo/master/examples/workflow-template/hello-world.yam
```