# EnvFromSource

EnvFromSource represents the source of a set of ConfigMaps

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config_map_ref** | [**ConfigMapEnvSource**](ConfigMapEnvSource.md) |  | [optional] 
**prefix** | **str** | An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER. | [optional] 
**secret_ref** | [**SecretEnvSource**](SecretEnvSource.md) |  | [optional] 

## Example

```python
from argo_workflows.models.env_from_source import EnvFromSource

# TODO update the JSON string below
json = "{}"
# create an instance of EnvFromSource from a JSON string
env_from_source_instance = EnvFromSource.from_json(json)
# print the JSON string representation of the object
print(EnvFromSource.to_json())

# convert the object into a dict
env_from_source_dict = env_from_source_instance.to_dict()
# create an instance of EnvFromSource from a dict
env_from_source_form_dict = env_from_source.from_dict(env_from_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


