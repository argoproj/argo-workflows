# IoArgoprojWorkflowV1alpha1StopStrategy

StopStrategy defines if the cron workflow will stop being triggered once a certain condition has been reached, involving a number of runs of the workflow

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**condition** | **str** | Condition defines a condition that stops scheduling workflows when evaluates to true. Use the keywords &#x60;failed&#x60; or &#x60;succeeded&#x60; to access the number of failed or successful child workflows. | 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


