# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1HTTPTrigger


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**basic_auth** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BasicAuth**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BasicAuth.md) |  | [optional] 
**headers** | **{str: (str,)}** |  | [optional] 
**method** | **str** |  | [optional] 
**parameters** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) | Parameters is the list of key-value extracted from event&#39;s payload that are applied to the HTTP trigger resource. | [optional] 
**payload** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) |  | [optional] 
**secure_headers** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SecureHeader]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SecureHeader.md) |  | [optional] 
**timeout** | **str** |  | [optional] 
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  | [optional] 
**url** | **str** | URL refers to the URL to send HTTP request to. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


