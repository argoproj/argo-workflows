# IoArgoprojEventsV1alpha1WebhookContext

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**endpoint** | Option<**String**> |  | [optional]
**max_payload_size** | Option<**String**> |  | [optional]
**metadata** | Option<**::std::collections::HashMap<String, String>**> |  | [optional]
**method** | Option<**String**> |  | [optional]
**port** | Option<**String**> | Port on which HTTP server is listening for incoming events. | [optional]
**server_cert_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**server_key_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**url** | Option<**String**> | URL is the url of the server. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


