

# IoArgoprojWorkflowV1alpha1CronWorkflowSpec

CronWorkflowSpec is the specification of a CronWorkflow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**concurrencyPolicy** | **String** | ConcurrencyPolicy is the K8s-style concurrency policy that will be used |  [optional]
**failedJobsHistoryLimit** | **Integer** | FailedJobsHistoryLimit is the number of failed jobs to be kept at a time |  [optional]
**schedule** | **String** | Schedule is a schedule to run the Workflow in Cron format | 
**schedules** | **List&lt;String&gt;** | Schedules is a list of schedules to run the Workflow in Cron format |  [optional]
**startingDeadlineSeconds** | **Integer** | StartingDeadlineSeconds is the K8s-style deadline that will limit the time a CronWorkflow will be run after its original scheduled time if it is missed. |  [optional]
**stopStrategy** | [**IoArgoprojWorkflowV1alpha1StopStrategy**](IoArgoprojWorkflowV1alpha1StopStrategy.md) |  |  [optional]
**successfulJobsHistoryLimit** | **Integer** | SuccessfulJobsHistoryLimit is the number of successful jobs to be kept at a time |  [optional]
**suspend** | **Boolean** | Suspend is a flag that will stop new CronWorkflows from running if set to true |  [optional]
**timezone** | **String** | Timezone is the timezone against which the cron schedule will be calculated, e.g. \&quot;Asia/Tokyo\&quot;. Default is machine&#39;s local time. |  [optional]
**workflowMetadata** | [**io.kubernetes.client.openapi.models.V1ObjectMeta**](io.kubernetes.client.openapi.models.V1ObjectMeta.md) |  |  [optional]
**workflowSpec** | [**IoArgoprojWorkflowV1alpha1WorkflowSpec**](IoArgoprojWorkflowV1alpha1WorkflowSpec.md) |  | 



