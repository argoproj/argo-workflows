# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BitbucketServerEventSource


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_token** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**bitbucketserver_base_url** | **str** | BitbucketServerBaseURL is the base URL for API requests to a custom endpoint. | [optional] 
**check_interval** | **str** |  | [optional] 
**delete_hook_on_finish** | **bool** |  | [optional] 
**events** | **[str]** |  | [optional] 
**filter** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**metadata** | **{str: (str,)}** |  | [optional] 
**one_event_per_change** | **bool** |  | [optional] 
**project_key** | **str** |  | [optional] 
**projects** | **[str]** |  | [optional] 
**repositories** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BitbucketServerRepository]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BitbucketServerRepository.md) |  | [optional] 
**repository_slug** | **str** |  | [optional] 
**skip_branch_refs_changed_on_open_pr** | **bool** |  | [optional] 
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  | [optional] 
**webhook** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1WebhookContext**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1WebhookContext.md) |  | [optional] 
**webhook_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


