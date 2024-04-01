# EventsourceUpdateEventSourceRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**event_source** | [**IoArgoprojEventsV1alpha1EventSource**](IoArgoprojEventsV1alpha1EventSource.md) |  | [optional] 
**name** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.eventsource_update_event_source_request import EventsourceUpdateEventSourceRequest

# TODO update the JSON string below
json = "{}"
# create an instance of EventsourceUpdateEventSourceRequest from a JSON string
eventsource_update_event_source_request_instance = EventsourceUpdateEventSourceRequest.from_json(json)
# print the JSON string representation of the object
print(EventsourceUpdateEventSourceRequest.to_json())

# convert the object into a dict
eventsource_update_event_source_request_dict = eventsource_update_event_source_request_instance.to_dict()
# create an instance of EventsourceUpdateEventSourceRequest from a dict
eventsource_update_event_source_request_form_dict = eventsource_update_event_source_request.from_dict(eventsource_update_event_source_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


