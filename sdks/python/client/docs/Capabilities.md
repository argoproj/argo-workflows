# Capabilities

Adds and removes POSIX capabilities from running containers.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**add** | **List[str]** | Added capabilities | [optional] 
**drop** | **List[str]** | Removed capabilities | [optional] 

## Example

```python
from argo_workflows.models.capabilities import Capabilities

# TODO update the JSON string below
json = "{}"
# create an instance of Capabilities from a JSON string
capabilities_instance = Capabilities.from_json(json)
# print the JSON string representation of the object
print(Capabilities.to_json())

# convert the object into a dict
capabilities_dict = capabilities_instance.to_dict()
# create an instance of Capabilities from a dict
capabilities_form_dict = capabilities.from_dict(capabilities_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


