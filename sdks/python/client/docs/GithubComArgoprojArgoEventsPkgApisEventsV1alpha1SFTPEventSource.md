# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SFTPEventSource

SFTPEventSource describes an event-source for sftp related events.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**address** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**event_type** | **str** |  | [optional] 
**filter** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**metadata** | **{str: (str,)}** |  | [optional] 
**password** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**poll_interval_duration** | **str** |  | [optional] 
**ssh_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**username** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**watch_path_config** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1WatchPathConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1WatchPathConfig.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


