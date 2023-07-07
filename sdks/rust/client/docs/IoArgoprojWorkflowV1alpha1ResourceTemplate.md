# IoArgoprojWorkflowV1alpha1ResourceTemplate

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**action** | **String** | Action is the action to perform to the resource. Must be one of: get, create, apply, delete, replace, patch | 
**failure_condition** | Option<**String**> | FailureCondition is a label selector expression which describes the conditions of the k8s resource in which the step was considered failed | [optional]
**flags** | Option<**Vec<String>**> | Flags is a set of additional options passed to kubectl before submitting a resource I.e. to disable resource validation: flags: [  \"--validate=false\"  # disable resource validation ] | [optional]
**manifest** | Option<**String**> | Manifest contains the kubernetes manifest | [optional]
**manifest_from** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ManifestFrom**](io.argoproj.workflow.v1alpha1.ManifestFrom.md)> |  | [optional]
**merge_strategy** | Option<**String**> | MergeStrategy is the strategy used to merge a patch. It defaults to \"strategic\" Must be one of: strategic, merge, json | [optional]
**set_owner_reference** | Option<**bool**> | SetOwnerReference sets the reference to the workflow on the OwnerReference of generated resource. | [optional]
**success_condition** | Option<**String**> | SuccessCondition is a label selector expression which describes the conditions of the k8s resource in which it is acceptable to proceed to the following step | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


