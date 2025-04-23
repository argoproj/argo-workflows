# IoArgoprojEventsV1alpha1SlackTrigger

SlackTrigger refers to the specification of the slack notification trigger.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**attachments** | **str** |  | [optional] 
**blocks** | **str** |  | [optional] 
**channel** | **str** |  | [optional] 
**message** | **str** |  | [optional] 
**parameters** | [**[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**sender** | [**IoArgoprojEventsV1alpha1SlackSender**](IoArgoprojEventsV1alpha1SlackSender.md) |  | [optional] 
**slack_token** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**thread** | [**IoArgoprojEventsV1alpha1SlackThread**](IoArgoprojEventsV1alpha1SlackThread.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


