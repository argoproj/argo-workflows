# IoArgoprojEventsV1alpha1OpenWhiskTrigger

OpenWhiskTrigger refers to the specification of the OpenWhisk trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**action_name** | **str** | Name of the action/function. | [optional] 
**auth_token** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**host** | **str** | Host URL of the OpenWhisk. | [optional] 
**namespace** | **str** | Namespace for the action. Defaults to \&quot;_\&quot;. +optional. | [optional] 
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**payload** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional] 
**version** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_open_whisk_trigger import IoArgoprojEventsV1alpha1OpenWhiskTrigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1OpenWhiskTrigger from a JSON string
io_argoproj_events_v1alpha1_open_whisk_trigger_instance = IoArgoprojEventsV1alpha1OpenWhiskTrigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1OpenWhiskTrigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_open_whisk_trigger_dict = io_argoproj_events_v1alpha1_open_whisk_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1OpenWhiskTrigger from a dict
io_argoproj_events_v1alpha1_open_whisk_trigger_form_dict = io_argoproj_events_v1alpha1_open_whisk_trigger.from_dict(io_argoproj_events_v1alpha1_open_whisk_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


