# EventsourceEventSourceWatchEvent


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**object** | [**IoArgoprojEventsV1alpha1EventSource**](IoArgoprojEventsV1alpha1EventSource.md) |  | [optional] 
**type** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.eventsource_event_source_watch_event import EventsourceEventSourceWatchEvent

# TODO update the JSON string below
json = "{}"
# create an instance of EventsourceEventSourceWatchEvent from a JSON string
eventsource_event_source_watch_event_instance = EventsourceEventSourceWatchEvent.from_json(json)
# print the JSON string representation of the object
print(EventsourceEventSourceWatchEvent.to_json())

# convert the object into a dict
eventsource_event_source_watch_event_dict = eventsource_event_source_watch_event_instance.to_dict()
# create an instance of EventsourceEventSourceWatchEvent from a dict
eventsource_event_source_watch_event_form_dict = eventsource_event_source_watch_event.from_dict(eventsource_event_source_watch_event_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


