# IoArgoprojEventsV1alpha1RedisStreamEventSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**consumer_group** | Option<**String**> |  | [optional]
**db** | Option<**i32**> |  | [optional]
**filter** | Option<[**crate::models::IoArgoprojEventsV1alpha1EventSourceFilter**](io.argoproj.events.v1alpha1.EventSourceFilter.md)> |  | [optional]
**host_address** | Option<**String**> |  | [optional]
**max_msg_count_per_read** | Option<**i32**> |  | [optional]
**metadata** | Option<**::std::collections::HashMap<String, String>**> |  | [optional]
**password** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**streams** | Option<**Vec<String>**> | Streams to look for entries. XREADGROUP is used on all streams using a single consumer group. | [optional]
**tls** | Option<[**crate::models::IoArgoprojEventsV1alpha1TlsConfig**](io.argoproj.events.v1alpha1.TLSConfig.md)> |  | [optional]
**username** | Option<**String**> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


