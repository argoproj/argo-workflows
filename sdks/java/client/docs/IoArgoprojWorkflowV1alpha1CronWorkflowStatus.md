

# IoArgoprojWorkflowV1alpha1CronWorkflowStatus

CronWorkflowStatus is the status of a CronWorkflow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active** | [**List&lt;io.kubernetes.client.openapi.models.V1ObjectReference&gt;**](io.kubernetes.client.openapi.models.V1ObjectReference.md) | Active is a list of active workflows stemming from this CronWorkflow | 
**conditions** | [**List&lt;IoArgoprojWorkflowV1alpha1Condition&gt;**](IoArgoprojWorkflowV1alpha1Condition.md) | Conditions is a list of conditions the CronWorkflow may have | 
**failed** | **Integer** | Failed is a counter of how many times a child workflow terminated in failed or errored state | 
**lastScheduledTime** | **java.time.Instant** |  | 
**phase** | **String** | Phase defines the cron workflow phase. It is changed to Stopped when the stopping condition is achieved which stops new CronWorkflows from running | 
**succeeded** | **Integer** | Succeeded is a counter of how many times the child workflows had success | 



