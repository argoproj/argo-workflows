# IoArgoprojEventsV1alpha1SensorStatus

SensorStatus contains information about the status of a sensor.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**status** | [**IoArgoprojEventsV1alpha1Status**](IoArgoprojEventsV1alpha1Status.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_sensor_status import IoArgoprojEventsV1alpha1SensorStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1SensorStatus from a JSON string
io_argoproj_events_v1alpha1_sensor_status_instance = IoArgoprojEventsV1alpha1SensorStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1SensorStatus.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_sensor_status_dict = io_argoproj_events_v1alpha1_sensor_status_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1SensorStatus from a dict
io_argoproj_events_v1alpha1_sensor_status_form_dict = io_argoproj_events_v1alpha1_sensor_status.from_dict(io_argoproj_events_v1alpha1_sensor_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


