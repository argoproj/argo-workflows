# IoArgoprojEventsV1alpha1EventPersistence


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**catchup** | [**IoArgoprojEventsV1alpha1CatchupConfiguration**](IoArgoprojEventsV1alpha1CatchupConfiguration.md) |  | [optional] 
**config_map** | [**IoArgoprojEventsV1alpha1ConfigMapPersistence**](IoArgoprojEventsV1alpha1ConfigMapPersistence.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_event_persistence import IoArgoprojEventsV1alpha1EventPersistence

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1EventPersistence from a JSON string
io_argoproj_events_v1alpha1_event_persistence_instance = IoArgoprojEventsV1alpha1EventPersistence.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1EventPersistence.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_event_persistence_dict = io_argoproj_events_v1alpha1_event_persistence_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1EventPersistence from a dict
io_argoproj_events_v1alpha1_event_persistence_form_dict = io_argoproj_events_v1alpha1_event_persistence.from_dict(io_argoproj_events_v1alpha1_event_persistence_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


