

# IoArgoprojEventsV1alpha1NATSTrigger

NATSTrigger refers to the specification of the NATS trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**parameters** | [**List&lt;IoArgoprojEventsV1alpha1TriggerParameter&gt;**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  |  [optional]
**payload** | [**List&lt;IoArgoprojEventsV1alpha1TriggerParameter&gt;**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  |  [optional]
**subject** | **String** | Name of the subject to put message on. |  [optional]
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  |  [optional]
**url** | **String** | URL of the NATS cluster. |  [optional]



