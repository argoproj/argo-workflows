# IoArgoprojWorkflowV1alpha1MutexHolding

MutexHolding describes the mutex and the object which is holding it.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**holder** | **str** | Holder is a reference to the object which holds the Mutex. Holding Scenario:   1. Current workflow&#39;s NodeID which is holding the lock.      e.g: ${NodeID} Waiting Scenario:   1. Current workflow or other workflow NodeID which is holding the lock.      e.g: ${WorkflowName}/${NodeID} | [optional] 
**mutex** | **str** | Reference for the mutex e.g: ${namespace}/mutex/${mutexName} | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


