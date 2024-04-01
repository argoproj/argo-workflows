# Sysctl

Sysctl defines a kernel parameter to be set

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name of a property to set | 
**value** | **str** | Value of a property to set | 

## Example

```python
from argo_workflows.models.sysctl import Sysctl

# TODO update the JSON string below
json = "{}"
# create an instance of Sysctl from a JSON string
sysctl_instance = Sysctl.from_json(json)
# print the JSON string representation of the object
print(Sysctl.to_json())

# convert the object into a dict
sysctl_dict = sysctl_instance.to_dict()
# create an instance of Sysctl from a dict
sysctl_form_dict = sysctl.from_dict(sysctl_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


