# Submitting A Workflow Via Automation

![GA](assets/ga.svg)

> v2.8 and after

You may want to consider using [events](events.md) or [webhooks](webhooks.md) instead.

Firstly, to do any automation, you'll need an ([access token](access-token.md)). For this example, our role needs extra permissions:

```sh
kubectl patch role jenkins -p '{"rules": [{"apiGroups": ["argoproj.io"], "resources": ["workflowtemplates"], "verbs": ["get"]}, {"apiGroups": ["argoproj.io"], "resources": ["workflows"], "verbs": ["create", "list", "get", "update"]}]}'
``` 

Next, create a workflow template 

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: hello-argo
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: a
            template: whalesay
    - name: whalesay
      container:
        image: docker/whalesay:latest
```

You can submit this workflow via an CLI or the [Argo Server API](rest-api.md). 

Submit via CLI (note how I add a label to help identify it later on):

````sh
argo submit --from wftmpl/hello-argo -l workflows.argoproj.io/workflow-template=hello-argo
````

Or submit via API:

```sh
curl $ARGO_SERVER/api/v1/workflows/argo/submit \
  -fs \
  -H "Authorization: $ARGO_TOKEN" \
  -d '{"resourceKind": "WorkflowTemplate", "resourceName": "hello-argo", "submitOptions": {"labels": "workflows.argoproj.io/workflow-template=hello-argo"}}' 
```

You'll see that the workflow has been created:

```sh
argo list
NAME               STATUS    AGE   DURATION   PRIORITY
hello-argo-77m4l   Running   33s   33s        0
```

See also:

See also:

* [access token](access-token.md)
* [events](events.md)
* [webhooks](webhooks.md)
* [resuming a workflow via automation](resuming-workflow-via-automation.md)
* [one workflow submitting another](workflow-submitting-workflow.md)
* [async pattern](async-pattern.md)
