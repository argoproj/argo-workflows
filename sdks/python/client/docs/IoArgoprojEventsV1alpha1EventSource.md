# IoArgoprojEventsV1alpha1EventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**ObjectMeta**](ObjectMeta.md) |  | [optional] 
**spec** | [**IoArgoprojEventsV1alpha1EventSourceSpec**](IoArgoprojEventsV1alpha1EventSourceSpec.md) |  | [optional] 
**status** | [**IoArgoprojEventsV1alpha1EventSourceStatus**](IoArgoprojEventsV1alpha1EventSourceStatus.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_event_source import IoArgoprojEventsV1alpha1EventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1EventSource from a JSON string
io_argoproj_events_v1alpha1_event_source_instance = IoArgoprojEventsV1alpha1EventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1EventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_event_source_dict = io_argoproj_events_v1alpha1_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1EventSource from a dict
io_argoproj_events_v1alpha1_event_source_form_dict = io_argoproj_events_v1alpha1_event_source.from_dict(io_argoproj_events_v1alpha1_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


