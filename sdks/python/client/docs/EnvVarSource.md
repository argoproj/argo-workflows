# EnvVarSource

EnvVarSource represents a source for the value of an EnvVar.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config_map_key_ref** | [**ConfigMapKeySelector**](ConfigMapKeySelector.md) |  | [optional] 
**field_ref** | [**ObjectFieldSelector**](ObjectFieldSelector.md) |  | [optional] 
**resource_field_ref** | [**ResourceFieldSelector**](ResourceFieldSelector.md) |  | [optional] 
**secret_key_ref** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.env_var_source import EnvVarSource

# TODO update the JSON string below
json = "{}"
# create an instance of EnvVarSource from a JSON string
env_var_source_instance = EnvVarSource.from_json(json)
# print the JSON string representation of the object
print(EnvVarSource.to_json())

# convert the object into a dict
env_var_source_dict = env_var_source_instance.to_dict()
# create an instance of EnvVarSource from a dict
env_var_source_form_dict = env_var_source.from_dict(env_var_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


