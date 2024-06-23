

# IoArgoprojWorkflowV1alpha1CronWorkflowStatus

CronWorkflowStatus is the status of a CronWorkflow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active** | [**List&lt;io.kubernetes.client.openapi.models.V1ObjectReference&gt;**](io.kubernetes.client.openapi.models.V1ObjectReference.md) | Active is a list of active workflows stemming from this CronWorkflow | 
**conditions** | [**List&lt;IoArgoprojWorkflowV1alpha1Condition&gt;**](IoArgoprojWorkflowV1alpha1Condition.md) | Conditions is a list of conditions the CronWorkflow may have | 
**failed** | **Integer** | v3.6 and after: Failed counts how many times child workflows failed | 
**lastScheduledTime** | **java.time.Instant** |  | 
**phase** | **String** | v3.6 and after: Phase is an enum of Active or Stopped. It changes to Stopped when stopStrategy.condition is true | 
**succeeded** | **Integer** | v3.6 and after: Succeeded counts how many times child workflows succeeded | 



