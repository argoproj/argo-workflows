# IoArgoprojEventsV1alpha1Selector

Selector represents conditional operation to select K8s objects.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **str** |  | [optional] 
**operation** | **str** |  | [optional] 
**value** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_selector import IoArgoprojEventsV1alpha1Selector

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1Selector from a JSON string
io_argoproj_events_v1alpha1_selector_instance = IoArgoprojEventsV1alpha1Selector.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1Selector.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_selector_dict = io_argoproj_events_v1alpha1_selector_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1Selector from a dict
io_argoproj_events_v1alpha1_selector_form_dict = io_argoproj_events_v1alpha1_selector.from_dict(io_argoproj_events_v1alpha1_selector_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


