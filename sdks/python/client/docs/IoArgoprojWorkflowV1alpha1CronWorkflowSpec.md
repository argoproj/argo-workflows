# IoArgoprojWorkflowV1alpha1CronWorkflowSpec

CronWorkflowSpec is the specification of a CronWorkflow

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**schedule** | **str** | Schedule is a schedule to run the Workflow in Cron format | 
**workflow_spec** | [**IoArgoprojWorkflowV1alpha1WorkflowSpec**](IoArgoprojWorkflowV1alpha1WorkflowSpec.md) |  | 
**concurrency_policy** | **str** | ConcurrencyPolicy is the K8s-style concurrency policy that will be used | [optional] 
**failed_jobs_history_limit** | **int** | FailedJobsHistoryLimit is the number of failed jobs to be kept at a time | [optional] 
**schedules** | **[str]** | Schedules is a list of schedules to run the Workflow in Cron format | [optional] 
**starting_deadline_seconds** | **int** | StartingDeadlineSeconds is the K8s-style deadline that will limit the time a CronWorkflow will be run after its original scheduled time if it is missed. | [optional] 
**stop_strategy** | [**IoArgoprojWorkflowV1alpha1StopStrategy**](IoArgoprojWorkflowV1alpha1StopStrategy.md) |  | [optional] 
**successful_jobs_history_limit** | **int** | SuccessfulJobsHistoryLimit is the number of successful jobs to be kept at a time | [optional] 
**suspend** | **bool** | Suspend is a flag that will stop new CronWorkflows from running if set to true | [optional] 
**timezone** | **str** | Timezone is the timezone against which the cron schedule will be calculated, e.g. \&quot;Asia/Tokyo\&quot;. Default is machine&#39;s local time. | [optional] 
**when** | **str** | v3.6 and after: When clause can be used to determine a run should or shouldn&#39;t be scheduled. | [optional] 
**workflow_metadata** | [**ObjectMeta**](ObjectMeta.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


