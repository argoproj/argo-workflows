# IoArgoprojEventsV1alpha1CustomTrigger

CustomTrigger refers to the specification of the custom trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cert_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Parameters is the list of parameters that is applied to resolved custom trigger trigger object. | [optional] 
**payload** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional] 
**secure** | **bool** |  | [optional] 
**server_name_override** | **str** | ServerNameOverride for the secure connection between sensor and custom trigger gRPC server. | [optional] 
**server_url** | **str** |  | [optional] 
**spec** | **Dict[str, str]** | Spec is the custom trigger resource specification that custom trigger gRPC server knows how to interpret. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_custom_trigger import IoArgoprojEventsV1alpha1CustomTrigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1CustomTrigger from a JSON string
io_argoproj_events_v1alpha1_custom_trigger_instance = IoArgoprojEventsV1alpha1CustomTrigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1CustomTrigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_custom_trigger_dict = io_argoproj_events_v1alpha1_custom_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1CustomTrigger from a dict
io_argoproj_events_v1alpha1_custom_trigger_form_dict = io_argoproj_events_v1alpha1_custom_trigger.from_dict(io_argoproj_events_v1alpha1_custom_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


