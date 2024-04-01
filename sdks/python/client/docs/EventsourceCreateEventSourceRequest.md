# EventsourceCreateEventSourceRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**event_source** | [**IoArgoprojEventsV1alpha1EventSource**](IoArgoprojEventsV1alpha1EventSource.md) |  | [optional] 
**namespace** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.eventsource_create_event_source_request import EventsourceCreateEventSourceRequest

# TODO update the JSON string below
json = "{}"
# create an instance of EventsourceCreateEventSourceRequest from a JSON string
eventsource_create_event_source_request_instance = EventsourceCreateEventSourceRequest.from_json(json)
# print the JSON string representation of the object
print(EventsourceCreateEventSourceRequest.to_json())

# convert the object into a dict
eventsource_create_event_source_request_dict = eventsource_create_event_source_request_instance.to_dict()
# create an instance of EventsourceCreateEventSourceRequest from a dict
eventsource_create_event_source_request_form_dict = eventsource_create_event_source_request.from_dict(eventsource_create_event_source_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


