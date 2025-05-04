# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureServiceBusTrigger


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**connection_string** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**parameters** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) |  | [optional] 
**payload** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional] 
**queue_name** | **str** |  | [optional] 
**subscription_name** | **str** |  | [optional] 
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  | [optional] 
**topic_name** | **str** |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


