# StreamResultOfEventsourceEventSourceWatchEvent


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**error** | [**GrpcGatewayRuntimeStreamError**](GrpcGatewayRuntimeStreamError.md) |  | [optional] 
**result** | [**EventsourceEventSourceWatchEvent**](EventsourceEventSourceWatchEvent.md) |  | [optional] 

## Example

```python
from argo_workflows.models.stream_result_of_eventsource_event_source_watch_event import StreamResultOfEventsourceEventSourceWatchEvent

# TODO update the JSON string below
json = "{}"
# create an instance of StreamResultOfEventsourceEventSourceWatchEvent from a JSON string
stream_result_of_eventsource_event_source_watch_event_instance = StreamResultOfEventsourceEventSourceWatchEvent.from_json(json)
# print the JSON string representation of the object
print(StreamResultOfEventsourceEventSourceWatchEvent.to_json())

# convert the object into a dict
stream_result_of_eventsource_event_source_watch_event_dict = stream_result_of_eventsource_event_source_watch_event_instance.to_dict()
# create an instance of StreamResultOfEventsourceEventSourceWatchEvent from a dict
stream_result_of_eventsource_event_source_watch_event_form_dict = stream_result_of_eventsource_event_source_watch_event.from_dict(stream_result_of_eventsource_event_source_watch_event_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


