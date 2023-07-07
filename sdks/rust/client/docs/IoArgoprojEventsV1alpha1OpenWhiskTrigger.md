# IoArgoprojEventsV1alpha1OpenWhiskTrigger

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**action_name** | Option<**String**> | Name of the action/function. | [optional]
**auth_token** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**host** | Option<**String**> | Host URL of the OpenWhisk. | [optional]
**namespace** | Option<**String**> | Namespace for the action. Defaults to \"_\". +optional. | [optional]
**parameters** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> |  | [optional]
**payload** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1TriggerParameter>**](io.argoproj.events.v1alpha1.TriggerParameter.md)> | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional]
**version** | Option<**String**> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


