# IoArgoprojEventsV1alpha1CalendarEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**exclusion_dates** | **List[str]** | ExclusionDates defines the list of DATE-TIME exceptions for recurring events. | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**interval** | **str** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**persistence** | [**IoArgoprojEventsV1alpha1EventPersistence**](IoArgoprojEventsV1alpha1EventPersistence.md) |  | [optional] 
**schedule** | **str** |  | [optional] 
**timezone** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_calendar_event_source import IoArgoprojEventsV1alpha1CalendarEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1CalendarEventSource from a JSON string
io_argoproj_events_v1alpha1_calendar_event_source_instance = IoArgoprojEventsV1alpha1CalendarEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1CalendarEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_calendar_event_source_dict = io_argoproj_events_v1alpha1_calendar_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1CalendarEventSource from a dict
io_argoproj_events_v1alpha1_calendar_event_source_form_dict = io_argoproj_events_v1alpha1_calendar_event_source.from_dict(io_argoproj_events_v1alpha1_calendar_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


