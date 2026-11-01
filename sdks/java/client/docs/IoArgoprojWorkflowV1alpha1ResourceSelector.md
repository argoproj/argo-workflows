

# IoArgoprojWorkflowV1alpha1ResourceSelector

ResourceSelector identifies the resources an EnumFrom lists

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**apiVersion** | **String** | APIVersion of the resource, e.g. &#x60;v1&#x60; or &#x60;apps/v1&#x60; | 
**kind** | **String** | Kind of the resource, e.g. &#x60;Namespace&#x60; or &#x60;Deployment&#x60;. Secrets are not allowed. | 
**namespace** | **String** | Namespace to list in. Defaults to the namespace the workflow is submitted to, and is ignored for cluster-scoped kinds. It may reference other workflow parameters, so the list can depend on another parameter&#39;s value. |  [optional]



