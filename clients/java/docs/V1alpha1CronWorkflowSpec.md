

# V1alpha1CronWorkflowSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**concurrencyPolicy** | **String** |  |  [optional]
**failedJobsHistoryLimit** | **Integer** |  |  [optional]
**schedule** | **String** |  |  [optional]
**startingDeadlineSeconds** | **String** | StartingDeadlineSeconds is the K8s-style deadline that will limit the time a CronWorkflow will be run after its original scheduled time if it is missed. |  [optional]
**successfulJobsHistoryLimit** | **Integer** |  |  [optional]
**suspend** | **Boolean** |  |  [optional]
**timezone** | **String** | Timezone is the timezone against which the cron schedule will be calculated, e.g. \&quot;Asia/Tokyo\&quot;. Default is machine&#39;s local time. |  [optional]
**workflowSpec** | [**V1alpha1WorkflowSpec**](V1alpha1WorkflowSpec.md) |  |  [optional]



