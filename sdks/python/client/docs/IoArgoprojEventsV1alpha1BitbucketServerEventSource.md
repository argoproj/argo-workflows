# IoArgoprojEventsV1alpha1BitbucketServerEventSource

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_token** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**bitbucketserver_base_url** | **str** |  | [optional] 
**delete_hook_on_finish** | **bool** |  | [optional] 
**events** | **list[str]** |  | [optional] 
**metadata** | **dict(str, str)** |  | [optional] 
**project_key** | **str** |  | [optional] 
**repository_slug** | **str** |  | [optional] 
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  | [optional] 
**webhook_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


