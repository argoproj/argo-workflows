# EventSeries

EventSeries contain information on series of events, i.e. thing that was/is happening continuously for some time.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**count** | **int** | Number of occurrences in this series up to the last heartbeat time | [optional] 
**last_observed_time** | **datetime** | MicroTime is version of Time with microsecond level precision. | [optional] 

## Example

```python
from argo_workflows.models.event_series import EventSeries

# TODO update the JSON string below
json = "{}"
# create an instance of EventSeries from a JSON string
event_series_instance = EventSeries.from_json(json)
# print the JSON string representation of the object
print(EventSeries.to_json())

# convert the object into a dict
event_series_dict = event_series_instance.to_dict()
# create an instance of EventSeries from a dict
event_series_form_dict = event_series.from_dict(event_series_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


