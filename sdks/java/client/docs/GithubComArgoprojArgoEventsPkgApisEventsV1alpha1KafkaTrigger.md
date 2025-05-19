

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaTrigger

KafkaTrigger refers to the specification of the Kafka trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**compress** | **Boolean** |  |  [optional]
**flushFrequency** | **Integer** |  |  [optional]
**headers** | **Map&lt;String, String&gt;** |  |  [optional]
**parameters** | [**List&lt;GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) | Parameters is the list of parameters that is applied to resolved Kafka trigger object. |  [optional]
**partition** | **Integer** |  |  [optional]
**partitioningKey** | **String** | The partitioning key for the messages put on the Kafka topic. +optional. |  [optional]
**payload** | [**List&lt;GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. |  [optional]
**requiredAcks** | **Integer** | RequiredAcks used in producer to tell the broker how many replica acknowledgements Defaults to 1 (Only wait for the leader to ack). +optional. |  [optional]
**sasl** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SASLConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SASLConfig.md) |  |  [optional]
**schemaRegistry** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SchemaRegistryConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SchemaRegistryConfig.md) |  |  [optional]
**secureHeaders** | [**List&lt;GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SecureHeader&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SecureHeader.md) |  |  [optional]
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  |  [optional]
**topic** | **String** |  |  [optional]
**url** | **String** | URL of the Kafka broker, multiple URLs separated by comma. |  [optional]
**version** | **String** |  |  [optional]



