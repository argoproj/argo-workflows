# IoArgoprojWorkflowV1alpha1CronWorkflowStatus

CronWorkflowStatus is the status of a CronWorkflow

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active** | [**[ObjectReference]**](ObjectReference.md) | Active is a list of active workflows stemming from this CronWorkflow | 
**conditions** | [**[IoArgoprojWorkflowV1alpha1Condition]**](IoArgoprojWorkflowV1alpha1Condition.md) | Conditions is a list of conditions the CronWorkflow may have | 
**failed** | **int** | v3.6 and after: Failed counts how many times child workflows failed | 
**last_scheduled_time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | 
**phase** | **str** | v3.6 and after: Phase is an enum of Active or Stopped. It changes to Stopped when stopStrategy.expression is true | 
**succeeded** | **int** | v3.6 and after: Succeeded counts how many times child workflows succeeded | 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


