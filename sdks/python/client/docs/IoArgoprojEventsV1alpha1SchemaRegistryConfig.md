# IoArgoprojEventsV1alpha1SchemaRegistryConfig


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**IoArgoprojEventsV1alpha1BasicAuth**](IoArgoprojEventsV1alpha1BasicAuth.md) |  | [optional] 
**schema_id** | **int** |  | [optional] 
**url** | **str** | Schema Registry URL. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_schema_registry_config import IoArgoprojEventsV1alpha1SchemaRegistryConfig

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1SchemaRegistryConfig from a JSON string
io_argoproj_events_v1alpha1_schema_registry_config_instance = IoArgoprojEventsV1alpha1SchemaRegistryConfig.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1SchemaRegistryConfig.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_schema_registry_config_dict = io_argoproj_events_v1alpha1_schema_registry_config_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1SchemaRegistryConfig from a dict
io_argoproj_events_v1alpha1_schema_registry_config_form_dict = io_argoproj_events_v1alpha1_schema_registry_config.from_dict(io_argoproj_events_v1alpha1_schema_registry_config_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


