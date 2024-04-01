# IoArgoprojEventsV1alpha1ResourceEventSource

ResourceEventSource refers to a event-source for K8s resource related events.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**event_types** | **List[str]** | EventTypes is the list of event type to watch. Possible values are - ADD, UPDATE and DELETE. | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1ResourceFilter**](IoArgoprojEventsV1alpha1ResourceFilter.md) |  | [optional] 
**group_version_resource** | [**GroupVersionResource**](GroupVersionResource.md) |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**namespace** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_resource_event_source import IoArgoprojEventsV1alpha1ResourceEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1ResourceEventSource from a JSON string
io_argoproj_events_v1alpha1_resource_event_source_instance = IoArgoprojEventsV1alpha1ResourceEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1ResourceEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_resource_event_source_dict = io_argoproj_events_v1alpha1_resource_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1ResourceEventSource from a dict
io_argoproj_events_v1alpha1_resource_event_source_form_dict = io_argoproj_events_v1alpha1_resource_event_source.from_dict(io_argoproj_events_v1alpha1_resource_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


