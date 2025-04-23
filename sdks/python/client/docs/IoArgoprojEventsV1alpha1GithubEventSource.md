# IoArgoprojEventsV1alpha1GithubEventSource


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active** | **bool** |  | [optional] 
**api_token** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**content_type** | **str** |  | [optional] 
**delete_hook_on_finish** | **bool** |  | [optional] 
**events** | **[str]** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**github_app** | [**IoArgoprojEventsV1alpha1GithubAppCreds**](IoArgoprojEventsV1alpha1GithubAppCreds.md) |  | [optional] 
**github_base_url** | **str** |  | [optional] 
**github_upload_url** | **str** |  | [optional] 
**id** | **str** |  | [optional] 
**insecure** | **bool** |  | [optional] 
**metadata** | **{str: (str,)}** |  | [optional] 
**organizations** | **[str]** | Organizations holds the names of organizations (used for organization level webhooks). Not required if Repositories is set. | [optional] 
**owner** | **str** |  | [optional] 
**repositories** | [**[IoArgoprojEventsV1alpha1OwnedRepositories]**](IoArgoprojEventsV1alpha1OwnedRepositories.md) | Repositories holds the information of repositories, which uses repo owner as the key, and list of repo names as the value. Not required if Organizations is set. | [optional] 
**repository** | **str** |  | [optional] 
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  | [optional] 
**webhook_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


