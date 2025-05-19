# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSTrigger

NATSTrigger refers to the specification of the NATS trigger.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSAuth**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSAuth.md) |  | [optional] 
**parameters** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) |  | [optional] 
**payload** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) |  | [optional] 
**subject** | **str** | Name of the subject to put message on. | [optional] 
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  | [optional] 
**url** | **str** | URL of the NATS cluster. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


