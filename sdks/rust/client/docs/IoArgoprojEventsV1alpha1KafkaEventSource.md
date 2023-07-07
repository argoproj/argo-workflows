# IoArgoprojEventsV1alpha1KafkaEventSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config** | Option<**String**> | Yaml format Sarama config for Kafka connection. It follows the struct of sarama.Config. See https://github.com/Shopify/sarama/blob/main/config.go e.g.  consumer:   fetch:     min: 1 net:   MaxOpenRequests: 5  +optional | [optional]
**connection_backoff** | Option<[**crate::models::IoArgoprojEventsV1alpha1Backoff**](io.argoproj.events.v1alpha1.Backoff.md)> |  | [optional]
**consumer_group** | Option<[**crate::models::IoArgoprojEventsV1alpha1KafkaConsumerGroup**](io.argoproj.events.v1alpha1.KafkaConsumerGroup.md)> |  | [optional]
**filter** | Option<[**crate::models::IoArgoprojEventsV1alpha1EventSourceFilter**](io.argoproj.events.v1alpha1.EventSourceFilter.md)> |  | [optional]
**json_body** | Option<**bool**> |  | [optional]
**limit_events_per_second** | Option<**String**> |  | [optional]
**metadata** | Option<**::std::collections::HashMap<String, String>**> |  | [optional]
**partition** | Option<**String**> |  | [optional]
**sasl** | Option<[**crate::models::IoArgoprojEventsV1alpha1SaslConfig**](io.argoproj.events.v1alpha1.SASLConfig.md)> |  | [optional]
**tls** | Option<[**crate::models::IoArgoprojEventsV1alpha1TlsConfig**](io.argoproj.events.v1alpha1.TLSConfig.md)> |  | [optional]
**topic** | Option<**String**> |  | [optional]
**url** | Option<**String**> |  | [optional]
**version** | Option<**String**> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


