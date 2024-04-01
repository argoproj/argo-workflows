# StreamResultOfEventsourceLogEntry


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**error** | [**GrpcGatewayRuntimeStreamError**](GrpcGatewayRuntimeStreamError.md) |  | [optional] 
**result** | [**EventsourceLogEntry**](EventsourceLogEntry.md) |  | [optional] 

## Example

```python
from argo_workflows.models.stream_result_of_eventsource_log_entry import StreamResultOfEventsourceLogEntry

# TODO update the JSON string below
json = "{}"
# create an instance of StreamResultOfEventsourceLogEntry from a JSON string
stream_result_of_eventsource_log_entry_instance = StreamResultOfEventsourceLogEntry.from_json(json)
# print the JSON string representation of the object
print(StreamResultOfEventsourceLogEntry.to_json())

# convert the object into a dict
stream_result_of_eventsource_log_entry_dict = stream_result_of_eventsource_log_entry_instance.to_dict()
# create an instance of StreamResultOfEventsourceLogEntry from a dict
stream_result_of_eventsource_log_entry_form_dict = stream_result_of_eventsource_log_entry.from_dict(stream_result_of_eventsource_log_entry_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


