

# IoArgoprojEventsV1alpha1KafkaTrigger

KafkaTrigger refers to the specification of the Kafka trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**compress** | **Boolean** |  |  [optional]
**flushFrequency** | **Integer** |  |  [optional]
**parameters** | [**List&lt;IoArgoprojEventsV1alpha1TriggerParameter&gt;**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Parameters is the list of parameters that is applied to resolved Kafka trigger object. |  [optional]
**partition** | **Integer** |  |  [optional]
**partitioningKey** | **String** | The partitioning key for the messages put on the Kafka topic. +optional. |  [optional]
**payload** | [**List&lt;IoArgoprojEventsV1alpha1TriggerParameter&gt;**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. |  [optional]
**requiredAcks** | **Integer** | RequiredAcks used in producer to tell the broker how many replica acknowledgements Defaults to 1 (Only wait for the leader to ack). +optional. |  [optional]
**sasl** | [**IoArgoprojEventsV1alpha1SASLConfig**](IoArgoprojEventsV1alpha1SASLConfig.md) |  |  [optional]
**schemaRegistry** | [**IoArgoprojEventsV1alpha1SchemaRegistryConfig**](IoArgoprojEventsV1alpha1SchemaRegistryConfig.md) |  |  [optional]
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  |  [optional]
**topic** | **String** |  |  [optional]
**url** | **String** | URL of the Kafka broker, multiple URLs separated by comma. |  [optional]
**version** | **String** |  |  [optional]



