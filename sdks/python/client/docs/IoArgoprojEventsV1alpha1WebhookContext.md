# IoArgoprojEventsV1alpha1WebhookContext

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**endpoint** | **str** |  | [optional] 
**metadata** | **dict(str, str)** |  | [optional] 
**method** | **str** |  | [optional] 
**port** | **str** | Port on which HTTP server is listening for incoming events. | [optional] 
**server_cert_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**server_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**url** | **str** | URL is the url of the server. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


