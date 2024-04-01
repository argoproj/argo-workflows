# ResourceRequirements

ResourceRequirements describes the compute resource requirements.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**limits** | **Dict[str, str]** | Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ | [optional] 
**requests** | **Dict[str, str]** | Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ | [optional] 

## Example

```python
from argo_workflows.models.resource_requirements import ResourceRequirements

# TODO update the JSON string below
json = "{}"
# create an instance of ResourceRequirements from a JSON string
resource_requirements_instance = ResourceRequirements.from_json(json)
# print the JSON string representation of the object
print(ResourceRequirements.to_json())

# convert the object into a dict
resource_requirements_dict = resource_requirements_instance.to_dict()
# create an instance of ResourceRequirements from a dict
resource_requirements_form_dict = resource_requirements.from_dict(resource_requirements_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


