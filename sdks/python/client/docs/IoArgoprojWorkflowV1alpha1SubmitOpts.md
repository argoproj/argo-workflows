# IoArgoprojWorkflowV1alpha1SubmitOpts

SubmitOpts are workflow submission options

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | **str** | Annotations adds to metadata.labels | [optional] 
**dry_run** | **bool** | DryRun validates the workflow on the client-side without creating it. This option is not supported in API | [optional] 
**entry_point** | **str** | Entrypoint overrides spec.entrypoint | [optional] 
**generate_name** | **str** | GenerateName overrides metadata.generateName | [optional] 
**labels** | **str** | Labels adds to metadata.labels | [optional] 
**name** | **str** | Name overrides metadata.name | [optional] 
**owner_reference** | [**OwnerReference**](OwnerReference.md) |  | [optional] 
**parameters** | **[str]** | Parameters passes input parameters to workflow | [optional] 
**pod_priority_class_name** | **str** | Set the podPriorityClassName of the workflow | [optional] 
**priority** | **int** | Priority is used if controller is configured to process limited number of workflows in parallel, higher priority workflows are processed first. | [optional] 
**server_dry_run** | **bool** | ServerDryRun validates the workflow on the server-side without creating it | [optional] 
**service_account** | **str** | ServiceAccount runs all pods in the workflow using specified ServiceAccount. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


