# IoArgoprojEventsV1alpha1AzureEventHubsTrigger

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fqdn** | Option<**String**> |  | [optional]
**hub_name** | Option<**String**> |  | [optional]
**parameters** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> |  | [optional]
**payload** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional]
**shared_access_key** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**shared_access_key_name** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


