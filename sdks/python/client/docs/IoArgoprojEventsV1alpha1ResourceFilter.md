# IoArgoprojEventsV1alpha1ResourceFilter


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**after_start** | **bool** |  | [optional] 
**created_by** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**fields** | [**List[IoArgoprojEventsV1alpha1Selector]**](IoArgoprojEventsV1alpha1Selector.md) |  | [optional] 
**labels** | [**List[IoArgoprojEventsV1alpha1Selector]**](IoArgoprojEventsV1alpha1Selector.md) |  | [optional] 
**prefix** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_resource_filter import IoArgoprojEventsV1alpha1ResourceFilter

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1ResourceFilter from a JSON string
io_argoproj_events_v1alpha1_resource_filter_instance = IoArgoprojEventsV1alpha1ResourceFilter.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1ResourceFilter.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_resource_filter_dict = io_argoproj_events_v1alpha1_resource_filter_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1ResourceFilter from a dict
io_argoproj_events_v1alpha1_resource_filter_form_dict = io_argoproj_events_v1alpha1_resource_filter.from_dict(io_argoproj_events_v1alpha1_resource_filter_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


