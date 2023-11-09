# IoArgoprojEventsV1alpha1AMQPEventSource


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**IoArgoprojEventsV1alpha1BasicAuth**](IoArgoprojEventsV1alpha1BasicAuth.md) |  | [optional] 
**connection_backoff** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  | [optional] 
**consume** | [**IoArgoprojEventsV1alpha1AMQPConsumeConfig**](IoArgoprojEventsV1alpha1AMQPConsumeConfig.md) |  | [optional] 
**exchange_declare** | [**IoArgoprojEventsV1alpha1AMQPExchangeDeclareConfig**](IoArgoprojEventsV1alpha1AMQPExchangeDeclareConfig.md) |  | [optional] 
**exchange_name** | **str** |  | [optional] 
**exchange_type** | **str** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **{str: (str,)}** |  | [optional] 
**queue_bind** | [**IoArgoprojEventsV1alpha1AMQPQueueBindConfig**](IoArgoprojEventsV1alpha1AMQPQueueBindConfig.md) |  | [optional] 
**queue_declare** | [**IoArgoprojEventsV1alpha1AMQPQueueDeclareConfig**](IoArgoprojEventsV1alpha1AMQPQueueDeclareConfig.md) |  | [optional] 
**routing_key** | **str** |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**url** | **str** |  | [optional] 
**url_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


