# ConfigMapEnvSource

ConfigMapEnvSource selects a ConfigMap to populate the environment variables with.  The contents of the target ConfigMap's Data field will represent the key-value pairs as environment variables.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names | [optional] 
**optional** | **bool** | Specify whether the ConfigMap must be defined | [optional] 

## Example

```python
from argo_workflows.models.config_map_env_source import ConfigMapEnvSource

# TODO update the JSON string below
json = "{}"
# create an instance of ConfigMapEnvSource from a JSON string
config_map_env_source_instance = ConfigMapEnvSource.from_json(json)
# print the JSON string representation of the object
print(ConfigMapEnvSource.to_json())

# convert the object into a dict
config_map_env_source_dict = config_map_env_source_instance.to_dict()
# create an instance of ConfigMapEnvSource from a dict
config_map_env_source_form_dict = config_map_env_source.from_dict(config_map_env_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


