# IoArgoprojWorkflowV1alpha1CronWorkflowSpec


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**concurrency_policy** | **str** |  | [optional] 
**failed_jobs_history_limit** | **int** |  | [optional] 
**schedule** | **str** |  | [optional] 
**starting_deadline_seconds** | **str** | StartingDeadlineSeconds is the K8s-style deadline that will limit the time a CronWorkflow will be run after its original scheduled time if it is missed. | [optional] 
**successful_jobs_history_limit** | **int** |  | [optional] 
**suspend** | **bool** |  | [optional] 
**timezone** | **str** | Timezone is the timezone against which the cron schedule will be calculated, e.g. \&quot;Asia/Tokyo\&quot;. Default is machine&#39;s local time. | [optional] 
**workflow_meta** | [**ObjectMeta**](ObjectMeta.md) |  | [optional] 
**workflow_spec** | [**IoArgoprojWorkflowV1alpha1WorkflowSpec**](IoArgoprojWorkflowV1alpha1WorkflowSpec.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


