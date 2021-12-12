# IoArgoprojWorkflowV1alpha1CronWorkflowStatus

CronWorkflowStatus is the status of a CronWorkflow
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active** | [**list[ObjectReference]**](ObjectReference.md) | Active is a list of active workflows stemming from this CronWorkflow | 
**conditions** | [**list[IoArgoprojWorkflowV1alpha1Condition]**](IoArgoprojWorkflowV1alpha1Condition.md) | Conditions is a list of conditions the CronWorkflow may have | 
**last_scheduled_time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


