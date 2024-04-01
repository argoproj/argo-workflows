# SensorCreateSensorRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**create_options** | [**CreateOptions**](CreateOptions.md) |  | [optional] 
**namespace** | **str** |  | [optional] 
**sensor** | [**IoArgoprojEventsV1alpha1Sensor**](IoArgoprojEventsV1alpha1Sensor.md) |  | [optional] 

## Example

```python
from argo_workflows.models.sensor_create_sensor_request import SensorCreateSensorRequest

# TODO update the JSON string below
json = "{}"
# create an instance of SensorCreateSensorRequest from a JSON string
sensor_create_sensor_request_instance = SensorCreateSensorRequest.from_json(json)
# print the JSON string representation of the object
print(SensorCreateSensorRequest.to_json())

# convert the object into a dict
sensor_create_sensor_request_dict = sensor_create_sensor_request_instance.to_dict()
# create an instance of SensorCreateSensorRequest from a dict
sensor_create_sensor_request_form_dict = sensor_create_sensor_request.from_dict(sensor_create_sensor_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


