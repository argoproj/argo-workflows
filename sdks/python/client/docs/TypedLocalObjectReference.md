# TypedLocalObjectReference

TypedLocalObjectReference contains enough information to let you locate the typed referenced object inside the same namespace.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_group** | **str** | APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required. | [optional] 
**kind** | **str** | Kind is the type of resource being referenced | 
**name** | **str** | Name is the name of resource being referenced | 

## Example

```python
from argo_workflows.models.typed_local_object_reference import TypedLocalObjectReference

# TODO update the JSON string below
json = "{}"
# create an instance of TypedLocalObjectReference from a JSON string
typed_local_object_reference_instance = TypedLocalObjectReference.from_json(json)
# print the JSON string representation of the object
print(TypedLocalObjectReference.to_json())

# convert the object into a dict
typed_local_object_reference_dict = typed_local_object_reference_instance.to_dict()
# create an instance of TypedLocalObjectReference from a dict
typed_local_object_reference_form_dict = typed_local_object_reference.from_dict(typed_local_object_reference_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


