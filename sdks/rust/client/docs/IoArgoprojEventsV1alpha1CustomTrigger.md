# IoArgoprojEventsV1alpha1CustomTrigger

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cert_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**parameters** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> | Parameters is the list of parameters that is applied to resolved custom trigger trigger object. | [optional]
**payload** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional]
**secure** | Option<**bool**> |  | [optional]
**server_name_override** | Option<**String**> | ServerNameOverride for the secure connection between sensor and custom trigger gRPC server. | [optional]
**server_url** | Option<**String**> |  | [optional]
**spec** | Option<**::std::collections::HashMap<String, String>**> | Spec is the custom trigger resource specification that custom trigger gRPC server knows how to interpret. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


