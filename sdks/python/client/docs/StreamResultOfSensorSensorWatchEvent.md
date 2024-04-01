# StreamResultOfSensorSensorWatchEvent


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**error** | [**GrpcGatewayRuntimeStreamError**](GrpcGatewayRuntimeStreamError.md) |  | [optional] 
**result** | [**SensorSensorWatchEvent**](SensorSensorWatchEvent.md) |  | [optional] 

## Example

```python
from argo_workflows.models.stream_result_of_sensor_sensor_watch_event import StreamResultOfSensorSensorWatchEvent

# TODO update the JSON string below
json = "{}"
# create an instance of StreamResultOfSensorSensorWatchEvent from a JSON string
stream_result_of_sensor_sensor_watch_event_instance = StreamResultOfSensorSensorWatchEvent.from_json(json)
# print the JSON string representation of the object
print(StreamResultOfSensorSensorWatchEvent.to_json())

# convert the object into a dict
stream_result_of_sensor_sensor_watch_event_dict = stream_result_of_sensor_sensor_watch_event_instance.to_dict()
# create an instance of StreamResultOfSensorSensorWatchEvent from a dict
stream_result_of_sensor_sensor_watch_event_form_dict = stream_result_of_sensor_sensor_watch_event.from_dict(stream_result_of_sensor_sensor_watch_event_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


