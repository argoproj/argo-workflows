# IoArgoprojEventsV1alpha1StorageGridEventSource

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_url** | **str** | APIURL is the url of the storagegrid api. | [optional] 
**auth_token** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**bucket** | **str** | Name of the bucket to register notifications for. | [optional] 
**events** | **list[str]** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1StorageGridFilter**](IoArgoprojEventsV1alpha1StorageGridFilter.md) |  | [optional] 
**metadata** | **dict(str, str)** |  | [optional] 
**region** | **str** |  | [optional] 
**topic_arn** | **str** |  | [optional] 
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


