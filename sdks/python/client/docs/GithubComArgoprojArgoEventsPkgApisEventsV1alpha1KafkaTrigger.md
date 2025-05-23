# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaTrigger

KafkaTrigger refers to the specification of the Kafka trigger.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**compress** | **bool** |  | [optional] 
**flush_frequency** | **int** |  | [optional] 
**headers** | **{str: (str,)}** |  | [optional] 
**parameters** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) | Parameters is the list of parameters that is applied to resolved Kafka trigger object. | [optional] 
**partition** | **int** |  | [optional] 
**partitioning_key** | **str** | The partitioning key for the messages put on the Kafka topic. +optional. | [optional] 
**payload** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional] 
**required_acks** | **int** | RequiredAcks used in producer to tell the broker how many replica acknowledgements Defaults to 1 (Only wait for the leader to ack). +optional. | [optional] 
**sasl** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SASLConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SASLConfig.md) |  | [optional] 
**schema_registry** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SchemaRegistryConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SchemaRegistryConfig.md) |  | [optional] 
**secure_headers** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SecureHeader]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SecureHeader.md) |  | [optional] 
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  | [optional] 
**topic** | **str** |  | [optional] 
**url** | **str** | URL of the Kafka broker, multiple URLs separated by comma. | [optional] 
**version** | **str** |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


