# IoArgoprojEventsV1alpha1HTTPTrigger

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**basic_auth** | [**IoArgoprojEventsV1alpha1BasicAuth**](IoArgoprojEventsV1alpha1BasicAuth.md) |  | [optional] 
**headers** | **dict(str, str)** |  | [optional] 
**method** | **str** |  | [optional] 
**parameters** | [**list[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Parameters is the list of key-value extracted from event&#39;s payload that are applied to the HTTP trigger resource. | [optional] 
**payload** | [**list[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**secure_headers** | [**list[IoArgoprojEventsV1alpha1SecureHeader]**](IoArgoprojEventsV1alpha1SecureHeader.md) |  | [optional] 
**timeout** | **str** |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**url** | **str** | URL refers to the URL to send HTTP request to. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


