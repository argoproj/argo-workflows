# IoArgoprojWorkflowV1alpha1CronWorkflowSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**concurrency_policy** | Option<**String**> | ConcurrencyPolicy is the K8s-style concurrency policy that will be used | [optional]
**failed_jobs_history_limit** | Option<**i32**> | FailedJobsHistoryLimit is the number of failed jobs to be kept at a time | [optional]
**schedule** | **String** | Schedule is a schedule to run the Workflow in Cron format | 
**starting_deadline_seconds** | Option<**i32**> | StartingDeadlineSeconds is the K8s-style deadline that will limit the time a CronWorkflow will be run after its original scheduled time if it is missed. | [optional]
**successful_jobs_history_limit** | Option<**i32**> | SuccessfulJobsHistoryLimit is the number of successful jobs to be kept at a time | [optional]
**suspend** | Option<**bool**> | Suspend is a flag that will stop new CronWorkflows from running if set to true | [optional]
**timezone** | Option<**String**> | Timezone is the timezone against which the cron schedule will be calculated, e.g. \"Asia/Tokyo\". Default is machine's local time. | [optional]
**workflow_metadata** | Option<[**crate::models::ObjectMeta**](ObjectMeta.md)> |  | [optional]
**workflow_spec** | [**crate::models::IoArgoprojWorkflowV1alpha1WorkflowSpec**](io.argoproj.workflow.v1alpha1.WorkflowSpec.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


