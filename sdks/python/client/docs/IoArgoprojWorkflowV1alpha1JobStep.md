# IoArgoprojWorkflowV1alpha1JobStep

JobStep is a step in a job

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name is the name of the step, must be unique within the job | 
**run** | **str** | Run is the shell script to run. | 
**_if** | **str** | If is the expression to evaluate to determine if the step should run, default \&quot;success()\&quot; | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


