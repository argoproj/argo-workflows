# Automation

![beta](assets/beta.svg)

> v2.8 and after

This is guide on automation. 

## Environment Variables

To do any automation you need to get the following environment variables:

* `ARGO_SERVER` - the hostname and port of your server, e.g. `argo-server:2746`
* `ARGO_TOKEN` - an [access token](access-token.md).

See `argo --help` to learn more.

## Waiting For External Events

For some workflows, you might want to wait for an external event. This can be achieved by using suspend nodes, and an HTTP request.

Use cases:

* One workflow depending on another workflow.
* Waiting for data to be available (e.g. in S3). 
* Resume a workflows from a CI pipeline. 

As an example, we'll create a workflow that waits for itself.

### Create A Workflow Template

Firstly, we need a workflow that waits for an event. We need to identify it using a label. A good way to do this is by using a workflow template, and any workflow created from the template will be labelled with the templates name:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: wait
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: a
            template: wait
    - name: wait
      suspend: {}
```

### Submit The Template

You can submit this workflow via an CLI or the [Argo Server API](rest-api.md), but you may need additional permissions to do so:

```shell script
kubectl patch role jenkins -p '{"rules": [{"apiGroups": ["argoproj.io"], "resources": ["workflowtemplates"], "verbs": ["get"]}, {"apiGroups": ["argoproj.io"], "resources": ["workflows"], "verbs": ["create", "list", "get", "update"]}]}'
``` 

````shell script
argo submit --from wftmpl/wait -l workflows.argoproj.io/workflow-template=wait
````

```shell script
curl $ARGO_SERVER/api/v1/workflows/argo/submit \
  -fs \
  -H "Authorization: Bearer $ARGO_TOKEN" \
  -d '{"resourceKind": "WorkflowTemplate", "resourceName": "wait", "submitOptions": {"labels": "workflows.argoproj.io/workflow-template=wait"}}' 
```

You'll see that the workflow has been created, and is now suspended waiting to be resumed.

```shell script
argo list
NAME         STATUS                AGE   DURATION   PRIORITY
wait-77m4l   Running (Suspended)   33s   33s        0
```

### Resume The Template

For automation, we want just the name of the workflow, we can use labels to get just this our suspended workflow:

```shell script
WF=$(argo list -l workflows.argoproj.io/workflow-template=wait --running -o name)
```

```shell script
WF=$(curl $ARGO_SERVER/api/v1/workflows/argo?listOptions.labelSelector=workflows.argoproj.io/workflow-template=wait,\!workflows.argoproj.io/completed \
  -fs \
  -H "Authorization: Bearer $ARGO_TOKEN" |
  jq -r '.items[0].metadata.name')
```

You can resume the workflow via the CLI or API too. If you have more than one node waiting, you must target it using a [node field selector](node-field-selector.md).

````shell script
argo resume $WF --node-field-selector displayName=a
````

```shell script
curl $ARGO_SERVER/api/v1/workflows/argo/$WF/resume \
  -fs \
  -X 'PUT' \
  -H "Authorization: Bearer $ARGO_TOKEN" \
  -d '{"nodeFieldSelector": "displayName=a"}' 
```

Now the workflow will have resumed and completed.

## One Workflow Starting Another Workflow

With Argo Server, you can do this using `curl`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: demo-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: a
            template: create-wf
    - name: create-wf
      script:
        image: curlimages/curl:latest
        command:
          - sh
        source: >
          curl http://argo-server:2746/api/v1/workflows/argo/submit \
            -fs \
            -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6Img5QmdsNDU5dFVWY3ZNbWVIdW1RQnlaZDNEUlR5SWJmYUtFWTl2TjRMaFUifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJhcmdvIiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZWNyZXQubmFtZSI6ImplbmtpbnMtdG9rZW4tbnNyeHAiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC5uYW1lIjoiamVua2lucyIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50LnVpZCI6IjA0ZjU4NmU2LTI3NTEtNDk3Yi04OTMxLWNjNGYwNTg0YTdjMCIsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDphcmdvOmplbmtpbnMifQ.IP8sluWZUNJob4mzGMALdqjaXSzq3-2oCmq14ey2GjnTsdp0irBXtrlrhlE43Wr0ZpsRrQi099ymnbTttTdTs4pZ-LaPjvzZz_7NRfGt2rKaAmakBmTBJdzGESKyy_mi-w92YLXPlK_6mn9pN6pCXHs80MGmkm4D_2VTGk1XiSUQeuLxdapJf6hbicurJqzDZrUtTihDxPdErmdBXOss4ZudX9DKxHaU4YOKuy_4aohKekY20HlXFtWGiBbJTLD2ZFMEZklmmHrb6Xfxl5Wu2tNj7QXfVAvB3PWg4ag9WlkqN5Hb4GwNrph_t8GTcsymzP9InENOAjCEtBmAto63Wg" \
            -d '{"resourceKind": "WorkflowTemplate", "resourceName": "wait", "submitOptions": {"labels": "workflows.argoproj.io/workflow-template=wait"}}' ```