# IoArgoprojEventsV1alpha1HttpTrigger

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**basic_auth** | Option<[**crate::models::IoArgoprojEventsV1alpha1BasicAuth**](io.argoproj.events.v1alpha1.BasicAuth.md)> |  | [optional]
**headers** | Option<**::std::collections::HashMap<String, String>**> |  | [optional]
**method** | Option<**String**> |  | [optional]
**parameters** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> | Parameters is the list of key-value extracted from event's payload that are applied to the HTTP trigger resource. | [optional]
**payload** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> |  | [optional]
**secure_headers** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1SecureHeader>**](io.argoproj.events.v1alpha1.SecureHeader.md)> |  | [optional]
**timeout** | Option<**String**> |  | [optional]
**tls** | Option<[**crate::models::IoArgoprojEventsV1alpha1TlsConfig**](io.argoproj.events.v1alpha1.TLSConfig.md)> |  | [optional]
**url** | Option<**String**> | URL refers to the URL to send HTTP request to. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


