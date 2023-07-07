# IoArgoprojEventsV1alpha1EmitterEventSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**broker** | Option<**String**> | Broker URI to connect to. | [optional]
**channel_key** | Option<**String**> |  | [optional]
**channel_name** | Option<**String**> |  | [optional]
**connection_backoff** | Option<[**crate::models::IoArgoprojEventsV1alpha1Backoff**](io.argoproj.events.v1alpha1.Backoff.md)> |  | [optional]
**filter** | Option<[**crate::models::IoArgoprojEventsV1alpha1EventSourceFilter**](io.argoproj.events.v1alpha1.EventSourceFilter.md)> |  | [optional]
**json_body** | Option<**bool**> |  | [optional]
**metadata** | Option<**::std::collections::HashMap<String, String>**> |  | [optional]
**password** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**tls** | Option<[**crate::models::IoArgoprojEventsV1alpha1TlsConfig**](io.argoproj.events.v1alpha1.TLSConfig.md)> |  | [optional]
**username** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


