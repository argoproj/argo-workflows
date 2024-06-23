# IoArgoprojWorkflowV1alpha1StopStrategy

v3.6 and after: StopStrategy defines if the CronWorkflow should stop scheduling based on a condition

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**condition** | **str** | v3.6 and after: Condition is an expression that stops scheduling workflows when true. Use the variables &#x60;failed&#x60; or &#x60;succeeded&#x60; to access the number of failed or successful child workflows. | 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


