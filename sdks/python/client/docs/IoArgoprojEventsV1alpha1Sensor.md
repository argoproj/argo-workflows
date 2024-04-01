# IoArgoprojEventsV1alpha1Sensor


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**ObjectMeta**](ObjectMeta.md) |  | [optional] 
**spec** | [**IoArgoprojEventsV1alpha1SensorSpec**](IoArgoprojEventsV1alpha1SensorSpec.md) |  | [optional] 
**status** | [**IoArgoprojEventsV1alpha1SensorStatus**](IoArgoprojEventsV1alpha1SensorStatus.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_sensor import IoArgoprojEventsV1alpha1Sensor

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1Sensor from a JSON string
io_argoproj_events_v1alpha1_sensor_instance = IoArgoprojEventsV1alpha1Sensor.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1Sensor.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_sensor_dict = io_argoproj_events_v1alpha1_sensor_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1Sensor from a dict
io_argoproj_events_v1alpha1_sensor_form_dict = io_argoproj_events_v1alpha1_sensor.from_dict(io_argoproj_events_v1alpha1_sensor_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


