# IoArgoprojEventsV1alpha1SensorSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**dependencies** | [**List[IoArgoprojEventsV1alpha1EventDependency]**](IoArgoprojEventsV1alpha1EventDependency.md) | Dependencies is a list of the events that this sensor is dependent on. | [optional] 
**error_on_failed_round** | **bool** | ErrorOnFailedRound if set to true, marks sensor state as &#x60;error&#x60; if the previous trigger round fails. Once sensor state is set to &#x60;error&#x60;, no further triggers will be processed. | [optional] 
**event_bus_name** | **str** |  | [optional] 
**logging_fields** | **Dict[str, str]** |  | [optional] 
**replicas** | **int** |  | [optional] 
**revision_history_limit** | **int** |  | [optional] 
**template** | [**IoArgoprojEventsV1alpha1Template**](IoArgoprojEventsV1alpha1Template.md) |  | [optional] 
**triggers** | [**List[IoArgoprojEventsV1alpha1Trigger]**](IoArgoprojEventsV1alpha1Trigger.md) | Triggers is a list of the things that this sensor evokes. These are the outputs from this sensor. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_sensor_spec import IoArgoprojEventsV1alpha1SensorSpec

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1SensorSpec from a JSON string
io_argoproj_events_v1alpha1_sensor_spec_instance = IoArgoprojEventsV1alpha1SensorSpec.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1SensorSpec.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_sensor_spec_dict = io_argoproj_events_v1alpha1_sensor_spec_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1SensorSpec from a dict
io_argoproj_events_v1alpha1_sensor_spec_form_dict = io_argoproj_events_v1alpha1_sensor_spec.from_dict(io_argoproj_events_v1alpha1_sensor_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


