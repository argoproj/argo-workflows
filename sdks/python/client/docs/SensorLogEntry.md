# SensorLogEntry


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**dependency_name** | **str** |  | [optional] 
**event_context** | **str** |  | [optional] 
**level** | **str** |  | [optional] 
**msg** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 
**sensor_name** | **str** |  | [optional] 
**time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**trigger_name** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.sensor_log_entry import SensorLogEntry

# TODO update the JSON string below
json = "{}"
# create an instance of SensorLogEntry from a JSON string
sensor_log_entry_instance = SensorLogEntry.from_json(json)
# print the JSON string representation of the object
print(SensorLogEntry.to_json())

# convert the object into a dict
sensor_log_entry_dict = sensor_log_entry_instance.to_dict()
# create an instance of SensorLogEntry from a dict
sensor_log_entry_form_dict = sensor_log_entry.from_dict(sensor_log_entry_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


