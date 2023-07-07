# IoArgoprojEventsV1alpha1KafkaTrigger

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**compress** | Option<**bool**> |  | [optional]
**flush_frequency** | Option<**i32**> |  | [optional]
**parameters** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> | Parameters is the list of parameters that is applied to resolved Kafka trigger object. | [optional]
**partition** | Option<**i32**> | Partition to write data to. | [optional]
**partitioning_key** | Option<**String**> | The partitioning key for the messages put on the Kafka topic. Defaults to broker url. +optional. | [optional]
**payload** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional]
**required_acks** | Option<**i32**> | RequiredAcks used in producer to tell the broker how many replica acknowledgements Defaults to 1 (Only wait for the leader to ack). +optional. | [optional]
**sasl** | Option<[**crate::models::IoArgoprojEventsV1alpha1SaslConfig**](io.argoproj.events.v1alpha1.SASLConfig.md)> |  | [optional]
**tls** | Option<[**crate::models::IoArgoprojEventsV1alpha1TlsConfig**](io.argoproj.events.v1alpha1.TLSConfig.md)> |  | [optional]
**topic** | Option<**String**> |  | [optional]
**url** | Option<**String**> | URL of the Kafka broker, multiple URLs separated by comma. | [optional]
**version** | Option<**String**> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


