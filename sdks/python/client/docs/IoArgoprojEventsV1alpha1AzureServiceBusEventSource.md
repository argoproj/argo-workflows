# IoArgoprojEventsV1alpha1AzureServiceBusEventSource


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**connection_string** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**fully_qualified_namespace** | **str** |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **{str: (str,)}** |  | [optional] 
**queue_name** | **str** |  | [optional] 
**subscription_name** | **str** |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**topic_name** | **str** |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


