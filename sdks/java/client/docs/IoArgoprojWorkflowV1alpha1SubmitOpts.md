

# IoArgoprojWorkflowV1alpha1SubmitOpts

SubmitOpts are workflow submission options

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | **String** | Annotations adds to metadata.labels |  [optional]
**dryRun** | **Boolean** | DryRun validates the workflow on the client-side without creating it. This option is not supported in API |  [optional]
**entryPoint** | **String** | Entrypoint overrides spec.entrypoint |  [optional]
**generateName** | **String** | GenerateName overrides metadata.generateName |  [optional]
**labels** | **String** | Labels adds to metadata.labels |  [optional]
**name** | **String** | Name overrides metadata.name |  [optional]
**ownerReference** | [**OwnerReference**](OwnerReference.md) |  |  [optional]
**parameters** | **List&lt;String&gt;** | Parameters passes input parameters to workflow |  [optional]
**podPriorityClassName** | **String** | Set the podPriorityClassName of the workflow |  [optional]
**priority** | **Integer** | Priority is used if controller is configured to process limited number of workflows in parallel, higher priority workflows are processed first. |  [optional]
**serverDryRun** | **Boolean** | ServerDryRun validates the workflow on the server-side without creating it |  [optional]
**serviceAccount** | **String** | ServiceAccount runs all pods in the workflow using specified ServiceAccount. |  [optional]



