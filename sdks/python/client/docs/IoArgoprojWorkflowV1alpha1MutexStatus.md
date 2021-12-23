# IoArgoprojWorkflowV1alpha1MutexStatus

MutexStatus contains which objects hold  mutex locks, and which objects this workflow is waiting on to release locks.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**holding** | [**[IoArgoprojWorkflowV1alpha1MutexHolding]**](IoArgoprojWorkflowV1alpha1MutexHolding.md) |  | [optional] 
**waiting** | [**[IoArgoprojWorkflowV1alpha1MutexHolding]**](IoArgoprojWorkflowV1alpha1MutexHolding.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


