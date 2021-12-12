# IoArgoprojEventsV1alpha1GenericEventSource

GenericEventSource refers to a generic event source. It can be used to implement a custom event source.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**config** | **str** |  | [optional] 
**insecure** | **bool** | Insecure determines the type of connection. | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **dict(str, str)** |  | [optional] 
**url** | **str** | URL of the gRPC server that implements the event source. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


