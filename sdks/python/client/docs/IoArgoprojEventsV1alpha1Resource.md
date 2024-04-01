# IoArgoprojEventsV1alpha1Resource

Resource represent arbitrary structured data.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**value** | **bytearray** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_resource import IoArgoprojEventsV1alpha1Resource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1Resource from a JSON string
io_argoproj_events_v1alpha1_resource_instance = IoArgoprojEventsV1alpha1Resource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1Resource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_resource_dict = io_argoproj_events_v1alpha1_resource_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1Resource from a dict
io_argoproj_events_v1alpha1_resource_form_dict = io_argoproj_events_v1alpha1_resource.from_dict(io_argoproj_events_v1alpha1_resource_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


