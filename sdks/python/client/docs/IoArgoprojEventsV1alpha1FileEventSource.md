# IoArgoprojEventsV1alpha1FileEventSource

FileEventSource describes an event-source for file related events.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**event_type** | **str** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**polling** | **bool** |  | [optional] 
**watch_path_config** | [**IoArgoprojEventsV1alpha1WatchPathConfig**](IoArgoprojEventsV1alpha1WatchPathConfig.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_file_event_source import IoArgoprojEventsV1alpha1FileEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1FileEventSource from a JSON string
io_argoproj_events_v1alpha1_file_event_source_instance = IoArgoprojEventsV1alpha1FileEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1FileEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_file_event_source_dict = io_argoproj_events_v1alpha1_file_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1FileEventSource from a dict
io_argoproj_events_v1alpha1_file_event_source_form_dict = io_argoproj_events_v1alpha1_file_event_source.from_dict(io_argoproj_events_v1alpha1_file_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


