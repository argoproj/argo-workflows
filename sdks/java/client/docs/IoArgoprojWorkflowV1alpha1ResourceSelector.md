

# IoArgoprojWorkflowV1alpha1ResourceSelector

ResourceSelector identifies the resources an EnumFrom lists

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**apiVersion** | **String** | APIVersion of the resource, for example &#x60;v1&#x60; or &#x60;apps/v1&#x60; | 
**kind** | **String** | Kind of the resource, for example &#x60;Namespace&#x60; or &#x60;Deployment&#x60;. The Argo Server never lists Secrets. | 
**namespace** | **String** | Namespace to list in. Defaults to the namespace you submit the workflow to. The Argo Server ignores it for cluster-scoped kinds. It may reference other workflow parameters, so the list can depend on another parameter&#39;s value. |  [optional]



