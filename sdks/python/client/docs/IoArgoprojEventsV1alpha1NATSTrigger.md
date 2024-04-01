# IoArgoprojEventsV1alpha1NATSTrigger

NATSTrigger refers to the specification of the NATS trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**payload** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**subject** | **str** | Name of the subject to put message on. | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**url** | **str** | URL of the NATS cluster. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_nats_trigger import IoArgoprojEventsV1alpha1NATSTrigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1NATSTrigger from a JSON string
io_argoproj_events_v1alpha1_nats_trigger_instance = IoArgoprojEventsV1alpha1NATSTrigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1NATSTrigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_nats_trigger_dict = io_argoproj_events_v1alpha1_nats_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1NATSTrigger from a dict
io_argoproj_events_v1alpha1_nats_trigger_form_dict = io_argoproj_events_v1alpha1_nats_trigger.from_dict(io_argoproj_events_v1alpha1_nats_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


