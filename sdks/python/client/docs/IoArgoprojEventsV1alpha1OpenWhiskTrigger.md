# IoArgoprojEventsV1alpha1OpenWhiskTrigger

OpenWhiskTrigger refers to the specification of the OpenWhisk trigger.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**action_name** | **str** | Name of the action/function. | [optional] 
**auth_token** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**host** | **str** | Host URL of the OpenWhisk. | [optional] 
**namespace** | **str** | Namespace for the action. Defaults to \&quot;_\&quot;. +optional. | [optional] 
**parameters** | [**list[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**payload** | [**list[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional] 
**version** | **str** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


