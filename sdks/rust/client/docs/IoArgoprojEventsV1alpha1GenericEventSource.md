# IoArgoprojEventsV1alpha1GenericEventSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**config** | Option<**String**> |  | [optional]
**filter** | Option<[**crate::models::IoArgoprojEventsV1alpha1EventSourceFilter**](io.argoproj.events.v1alpha1.EventSourceFilter.md)> |  | [optional]
**insecure** | Option<**bool**> | Insecure determines the type of connection. | [optional]
**json_body** | Option<**bool**> |  | [optional]
**metadata** | Option<**::std::collections::HashMap<String, String>**> |  | [optional]
**url** | Option<**String**> | URL of the gRPC server that implements the event source. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


