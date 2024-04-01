# IoArgoprojEventsV1alpha1SlackTrigger

SlackTrigger refers to the specification of the slack notification trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**attachments** | **str** |  | [optional] 
**blocks** | **str** |  | [optional] 
**channel** | **str** |  | [optional] 
**message** | **str** |  | [optional] 
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**sender** | [**IoArgoprojEventsV1alpha1SlackSender**](IoArgoprojEventsV1alpha1SlackSender.md) |  | [optional] 
**slack_token** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**thread** | [**IoArgoprojEventsV1alpha1SlackThread**](IoArgoprojEventsV1alpha1SlackThread.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_slack_trigger import IoArgoprojEventsV1alpha1SlackTrigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1SlackTrigger from a JSON string
io_argoproj_events_v1alpha1_slack_trigger_instance = IoArgoprojEventsV1alpha1SlackTrigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1SlackTrigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_slack_trigger_dict = io_argoproj_events_v1alpha1_slack_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1SlackTrigger from a dict
io_argoproj_events_v1alpha1_slack_trigger_form_dict = io_argoproj_events_v1alpha1_slack_trigger.from_dict(io_argoproj_events_v1alpha1_slack_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


