# IoArgoprojEventsV1alpha1StandardK8STrigger


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**live_object** | **bool** |  | [optional] 
**operation** | **str** |  | [optional] 
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Parameters is the list of parameters that is applied to resolved K8s trigger object. | [optional] 
**patch_strategy** | **str** |  | [optional] 
**source** | [**IoArgoprojEventsV1alpha1ArtifactLocation**](IoArgoprojEventsV1alpha1ArtifactLocation.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_standard_k8_s_trigger import IoArgoprojEventsV1alpha1StandardK8STrigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1StandardK8STrigger from a JSON string
io_argoproj_events_v1alpha1_standard_k8_s_trigger_instance = IoArgoprojEventsV1alpha1StandardK8STrigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1StandardK8STrigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_standard_k8_s_trigger_dict = io_argoproj_events_v1alpha1_standard_k8_s_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1StandardK8STrigger from a dict
io_argoproj_events_v1alpha1_standard_k8_s_trigger_form_dict = io_argoproj_events_v1alpha1_standard_k8_s_trigger.from_dict(io_argoproj_events_v1alpha1_standard_k8_s_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


