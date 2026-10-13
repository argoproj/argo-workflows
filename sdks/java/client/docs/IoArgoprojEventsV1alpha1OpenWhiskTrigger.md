

# IoArgoprojEventsV1alpha1OpenWhiskTrigger

OpenWhiskTrigger refers to the specification of the OpenWhisk trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**actionName** | **String** | Name of the action/function. |  [optional]
**authToken** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**host** | **String** | Host URL of the OpenWhisk. |  [optional]
**namespace** | **String** | Namespace for the action. Defaults to \&quot;_\&quot;. +optional. |  [optional]
**parameters** | [**List&lt;IoArgoprojEventsV1alpha1TriggerParameter&gt;**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  |  [optional]
**payload** | [**List&lt;IoArgoprojEventsV1alpha1TriggerParameter&gt;**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. |  [optional]
**version** | **String** |  |  [optional]



