# EventsourceLogEntry


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**event_name** | **str** |  | [optional] 
**event_source_name** | **str** |  | [optional] 
**event_source_type** | **str** |  | [optional] 
**level** | **str** |  | [optional] 
**msg** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 
**time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 

## Example

```python
from argo_workflows.models.eventsource_log_entry import EventsourceLogEntry

# TODO update the JSON string below
json = "{}"
# create an instance of EventsourceLogEntry from a JSON string
eventsource_log_entry_instance = EventsourceLogEntry.from_json(json)
# print the JSON string representation of the object
print(EventsourceLogEntry.to_json())

# convert the object into a dict
eventsource_log_entry_dict = eventsource_log_entry_instance.to_dict()
# create an instance of EventsourceLogEntry from a dict
eventsource_log_entry_form_dict = eventsource_log_entry.from_dict(eventsource_log_entry_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


