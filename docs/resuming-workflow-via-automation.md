# Resume A Workflow

For automation, we want just the name of the workflow, we can use labels to get just this our suspended workflow:

```sh
WF=$(argo list -l workflows.argoproj.io/workflow-template=wait --running -o name)
```

```sh
WF=$(curl $ARGO_SERVER/api/v1/workflows/argo?listOptions.labelSelector=workflows.argoproj.io/workflow-template=wait,\!workflows.argoproj.io/completed \
  -fs \
  -H "Authorization: $ARGO_TOKEN" |
  jq -r '.items[0].metadata.name')
```

You can resume the workflow via the CLI or API too. If you have more than one node waiting, you must target it using a [node field selector](node-field-selector.md).

````sh
argo resume $WF --node-field-selector displayName=a
````

```sh
curl $ARGO_SERVER/api/v1/workflows/argo/$WF/resume \
  -fs \
  -X 'PUT' \
  -H "Authorization: $ARGO_TOKEN" \
  -d '{"nodeFieldSelector": "displayName=a"}' 
```

Now the workflow will have resumed and completed.

See also:

* [access token](access-token.md)
* [resuming a workflow via automation](resuming-workflow-via-automation.md)
* [submitting a workflow via automation](submit-workflow-via-automation.md)
* [one workflow submitting another](workflow-submitting-workflow.md)
* [async pattern](async-pattern.md)
