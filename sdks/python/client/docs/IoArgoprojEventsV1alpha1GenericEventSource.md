# IoArgoprojEventsV1alpha1GenericEventSource

GenericEventSource refers to a generic event source. It can be used to implement a custom event source.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**config** | **str** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**insecure** | **bool** | Insecure determines the type of connection. | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **{str: (str,)}** |  | [optional] 
**url** | **str** | URL of the gRPC server that implements the event source. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


