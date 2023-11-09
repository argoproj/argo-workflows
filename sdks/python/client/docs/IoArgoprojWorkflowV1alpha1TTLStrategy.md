# IoArgoprojWorkflowV1alpha1TTLStrategy

TTLStrategy is the strategy for the time to live depending on if the workflow succeeded or failed

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**seconds_after_completion** | **int** | SecondsAfterCompletion is the number of seconds to live after completion | [optional] 
**seconds_after_failure** | **int** | SecondsAfterFailure is the number of seconds to live after failure | [optional] 
**seconds_after_success** | **int** | SecondsAfterSuccess is the number of seconds to live after success | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


