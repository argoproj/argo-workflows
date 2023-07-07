# IoArgoprojEventsV1alpha1AmqpEventSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | Option<[**crate::models::IoArgoprojEventsV1alpha1BasicAuth**](io.argoproj.events.v1alpha1.BasicAuth.md)> |  | [optional]
**connection_backoff** | Option<[**crate::models::IoArgoprojEventsV1alpha1Backoff**](io.argoproj.events.v1alpha1.Backoff.md)> |  | [optional]
**consume** | Option<[**crate::models::IoArgoprojEventsV1alpha1AmqpConsumeConfig**](io.argoproj.events.v1alpha1.AMQPConsumeConfig.md)> |  | [optional]
**exchange_declare** | Option<[**crate::models::IoArgoprojEventsV1alpha1AmqpExchangeDeclareConfig**](io.argoproj.events.v1alpha1.AMQPExchangeDeclareConfig.md)> |  | [optional]
**exchange_name** | Option<**String**> |  | [optional]
**exchange_type** | Option<**String**> |  | [optional]
**filter** | Option<[**crate::models::IoArgoprojEventsV1alpha1EventSourceFilter**](io.argoproj.events.v1alpha1.EventSourceFilter.md)> |  | [optional]
**json_body** | Option<**bool**> |  | [optional]
**metadata** | Option<**::std::collections::HashMap<String, String>**> |  | [optional]
**queue_bind** | Option<[**crate::models::IoArgoprojEventsV1alpha1AmqpQueueBindConfig**](io.argoproj.events.v1alpha1.AMQPQueueBindConfig.md)> |  | [optional]
**queue_declare** | Option<[**crate::models::IoArgoprojEventsV1alpha1AmqpQueueDeclareConfig**](io.argoproj.events.v1alpha1.AMQPQueueDeclareConfig.md)> |  | [optional]
**routing_key** | Option<**String**> |  | [optional]
**tls** | Option<[**crate::models::IoArgoprojEventsV1alpha1TlsConfig**](io.argoproj.events.v1alpha1.TLSConfig.md)> |  | [optional]
**url** | Option<**String**> |  | [optional]
**url_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


