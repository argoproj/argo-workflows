# LocalObjectReference

LocalObjectReference contains enough information to let you locate the referenced object inside the same namespace.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names | [optional] 

## Example

```python
from argo_workflows.models.local_object_reference import LocalObjectReference

# TODO update the JSON string below
json = "{}"
# create an instance of LocalObjectReference from a JSON string
local_object_reference_instance = LocalObjectReference.from_json(json)
# print the JSON string representation of the object
print(LocalObjectReference.to_json())

# convert the object into a dict
local_object_reference_dict = local_object_reference_instance.to_dict()
# create an instance of LocalObjectReference from a dict
local_object_reference_form_dict = local_object_reference.from_dict(local_object_reference_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


