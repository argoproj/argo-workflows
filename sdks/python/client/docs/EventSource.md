# EventSource

EventSource contains information for an event.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**component** | **str** | Component from which the event is generated. | [optional] 
**host** | **str** | Node name on which the event is generated. | [optional] 

## Example

```python
from argo_workflows.models.event_source import EventSource

# TODO update the JSON string below
json = "{}"
# create an instance of EventSource from a JSON string
event_source_instance = EventSource.from_json(json)
# print the JSON string representation of the object
print(EventSource.to_json())

# convert the object into a dict
event_source_dict = event_source_instance.to_dict()
# create an instance of EventSource from a dict
event_source_form_dict = event_source.from_dict(event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


