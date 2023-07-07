# IoArgoprojWorkflowV1alpha1SubmitOpts

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | Option<**String**> | Annotations adds to metadata.labels | [optional]
**dry_run** | Option<**bool**> | DryRun validates the workflow on the client-side without creating it. This option is not supported in API | [optional]
**entry_point** | Option<**String**> | Entrypoint overrides spec.entrypoint | [optional]
**generate_name** | Option<**String**> | GenerateName overrides metadata.generateName | [optional]
**labels** | Option<**String**> | Labels adds to metadata.labels | [optional]
**name** | Option<**String**> | Name overrides metadata.name | [optional]
**owner_reference** | Option<[**crate::models::OwnerReference**](OwnerReference.md)> |  | [optional]
**parameters** | Option<**Vec<String>**> | Parameters passes input parameters to workflow | [optional]
**pod_priority_class_name** | Option<**String**> | Set the podPriorityClassName of the workflow | [optional]
**priority** | Option<**i32**> | Priority is used if controller is configured to process limited number of workflows in parallel, higher priority workflows are processed first. | [optional]
**server_dry_run** | Option<**bool**> | ServerDryRun validates the workflow on the server-side without creating it | [optional]
**service_account** | Option<**String**> | ServiceAccount runs all pods in the workflow using specified ServiceAccount. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


