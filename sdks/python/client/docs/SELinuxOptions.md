# SELinuxOptions

SELinuxOptions are the labels to be applied to the container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**level** | **str** | Level is SELinux level label that applies to the container. | [optional] 
**role** | **str** | Role is a SELinux role label that applies to the container. | [optional] 
**type** | **str** | Type is a SELinux type label that applies to the container. | [optional] 
**user** | **str** | User is a SELinux user label that applies to the container. | [optional] 

## Example

```python
from argo_workflows.models.se_linux_options import SELinuxOptions

# TODO update the JSON string below
json = "{}"
# create an instance of SELinuxOptions from a JSON string
se_linux_options_instance = SELinuxOptions.from_json(json)
# print the JSON string representation of the object
print(SELinuxOptions.to_json())

# convert the object into a dict
se_linux_options_dict = se_linux_options_instance.to_dict()
# create an instance of SELinuxOptions from a dict
se_linux_options_form_dict = se_linux_options.from_dict(se_linux_options_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


