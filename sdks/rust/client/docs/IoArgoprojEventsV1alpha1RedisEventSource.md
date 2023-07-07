# IoArgoprojEventsV1alpha1RedisEventSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**channels** | Option<**Vec<String>**> |  | [optional]
**db** | Option<**i32**> |  | [optional]
**filter** | Option<[**crate::models::IoArgoprojEventsV1alpha1EventSourceFilter**](io.argoproj.events.v1alpha1.EventSourceFilter.md)> |  | [optional]
**host_address** | Option<**String**> |  | [optional]
**json_body** | Option<**bool**> |  | [optional]
**metadata** | Option<**::std::collections::HashMap<String, String>**> |  | [optional]
**namespace** | Option<**String**> |  | [optional]
**password** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**tls** | Option<[**crate::models::IoArgoprojEventsV1alpha1TlsConfig**](io.argoproj.events.v1alpha1.TLSConfig.md)> |  | [optional]
**username** | Option<**String**> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


