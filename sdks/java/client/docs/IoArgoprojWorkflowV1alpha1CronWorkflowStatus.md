

# IoArgoprojWorkflowV1alpha1CronWorkflowStatus

CronWorkflowStatus is the status of a CronWorkflow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active** | [**List&lt;io.kubernetes.client.openapi.models.V1ObjectReference&gt;**](io.kubernetes.client.openapi.models.V1ObjectReference.md) | Active is a list of active workflows stemming from this CronWorkflow |  [optional]
**conditions** | [**List&lt;IoArgoprojWorkflowV1alpha1Condition&gt;**](IoArgoprojWorkflowV1alpha1Condition.md) | Conditions is a list of conditions the CronWorkflow may have |  [optional]
**failed** | **Integer** | v3.6 and after: Failed counts how many times child workflows failed |  [optional]
**lastScheduledTime** | **java.time.Instant** |  |  [optional]
**phase** | **String** | v3.6 and after: Phase is an enum of Active or Stopped. It changes to Stopped when stopStrategy.expression is true |  [optional]
**resolvedSchedules** | **Map&lt;String, String&gt;** | ResolvedSchedules maps a schedule of the spec using a &#x60;H&#x60; (hash) token to what it resolved to. Set by the controller, schedules without a &#x60;H&#x60; are not listed |  [optional]
**succeeded** | **Integer** | v3.6 and after: Succeeded counts how many times child workflows succeeded |  [optional]



