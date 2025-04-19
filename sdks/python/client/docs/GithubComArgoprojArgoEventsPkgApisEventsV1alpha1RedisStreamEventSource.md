# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1RedisStreamEventSource


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**consumer_group** | **str** |  | [optional] 
**db** | **int** |  | [optional] 
**filter** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**host_address** | **str** |  | [optional] 
**max_msg_count_per_read** | **int** |  | [optional] 
**metadata** | **{str: (str,)}** |  | [optional] 
**password** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**streams** | **[str]** | Streams to look for entries. XREADGROUP is used on all streams using a single consumer group. | [optional] 
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  | [optional] 
**username** | **str** |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


