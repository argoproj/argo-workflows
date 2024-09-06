# IoArgoprojWorkflowV1alpha1SuspendTemplate

SuspendTemplate is a template subtype to suspend a workflow at a predetermined point in time

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**approvers** | **[str]** | List of approvers emails that are required to review the workflow before lifting the suspend. | [optional] 
**duration** | **str** | Duration is the seconds to wait before automatically resuming a template. Must be a string. Default unit is seconds. Could also be a Duration, e.g.: \&quot;2m\&quot;, \&quot;6h\&quot; | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


