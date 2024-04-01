# IoArgoprojEventsV1alpha1Condition


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**last_transition_time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**message** | **str** |  | [optional] 
**reason** | **str** |  | [optional] 
**status** | **str** |  | [optional] 
**type** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_condition import IoArgoprojEventsV1alpha1Condition

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1Condition from a JSON string
io_argoproj_events_v1alpha1_condition_instance = IoArgoprojEventsV1alpha1Condition.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1Condition.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_condition_dict = io_argoproj_events_v1alpha1_condition_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1Condition from a dict
io_argoproj_events_v1alpha1_condition_form_dict = io_argoproj_events_v1alpha1_condition.from_dict(io_argoproj_events_v1alpha1_condition_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


