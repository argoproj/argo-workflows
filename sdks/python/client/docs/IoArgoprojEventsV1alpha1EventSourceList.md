# IoArgoprojEventsV1alpha1EventSourceList


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[IoArgoprojEventsV1alpha1EventSource]**](IoArgoprojEventsV1alpha1EventSource.md) |  | [optional] 
**metadata** | [**ListMeta**](ListMeta.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_event_source_list import IoArgoprojEventsV1alpha1EventSourceList

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1EventSourceList from a JSON string
io_argoproj_events_v1alpha1_event_source_list_instance = IoArgoprojEventsV1alpha1EventSourceList.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1EventSourceList.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_event_source_list_dict = io_argoproj_events_v1alpha1_event_source_list_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1EventSourceList from a dict
io_argoproj_events_v1alpha1_event_source_list_form_dict = io_argoproj_events_v1alpha1_event_source_list.from_dict(io_argoproj_events_v1alpha1_event_source_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


