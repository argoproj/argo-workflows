# ConfigMapKeySelector

Selects a key from a ConfigMap.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **str** | The key to select. | 
**name** | **str** | Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names | [optional] 
**optional** | **bool** | Specify whether the ConfigMap or its key must be defined | [optional] 

## Example

```python
from argo_workflows.models.config_map_key_selector import ConfigMapKeySelector

# TODO update the JSON string below
json = "{}"
# create an instance of ConfigMapKeySelector from a JSON string
config_map_key_selector_instance = ConfigMapKeySelector.from_json(json)
# print the JSON string representation of the object
print(ConfigMapKeySelector.to_json())

# convert the object into a dict
config_map_key_selector_dict = config_map_key_selector_instance.to_dict()
# create an instance of ConfigMapKeySelector from a dict
config_map_key_selector_form_dict = config_map_key_selector.from_dict(config_map_key_selector_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


