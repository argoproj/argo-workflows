# IoArgoprojEventsV1alpha1StorageGridEventSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_url** | Option<**String**> | APIURL is the url of the storagegrid api. | [optional]
**auth_token** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**bucket** | Option<**String**> | Name of the bucket to register notifications for. | [optional]
**events** | Option<**Vec<String>**> |  | [optional]
**filter** | Option<[**crate::models::IoArgoprojEventsV1alpha1StorageGridFilter**](io.argoproj.events.v1alpha1.StorageGridFilter.md)> |  | [optional]
**metadata** | Option<**::std::collections::HashMap<String, String>**> |  | [optional]
**region** | Option<**String**> |  | [optional]
**topic_arn** | Option<**String**> |  | [optional]
**webhook** | Option<[**crate::models::IoArgoprojEventsV1alpha1WebhookContext**](io.argoproj.events.v1alpha1.WebhookContext.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


