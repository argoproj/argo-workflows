# SensorSensorWatchEvent


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**object** | [**IoArgoprojEventsV1alpha1Sensor**](IoArgoprojEventsV1alpha1Sensor.md) |  | [optional] 
**type** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.sensor_sensor_watch_event import SensorSensorWatchEvent

# TODO update the JSON string below
json = "{}"
# create an instance of SensorSensorWatchEvent from a JSON string
sensor_sensor_watch_event_instance = SensorSensorWatchEvent.from_json(json)
# print the JSON string representation of the object
print(SensorSensorWatchEvent.to_json())

# convert the object into a dict
sensor_sensor_watch_event_dict = sensor_sensor_watch_event_instance.to_dict()
# create an instance of SensorSensorWatchEvent from a dict
sensor_sensor_watch_event_form_dict = sensor_sensor_watch_event.from_dict(sensor_sensor_watch_event_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


