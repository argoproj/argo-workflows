# ObjectFieldSelector

ObjectFieldSelector selects an APIVersioned field of an object.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **str** | Version of the schema the FieldPath is written in terms of, defaults to \&quot;v1\&quot;. | [optional] 
**field_path** | **str** | Path of the field to select in the specified API version. | 

## Example

```python
from argo_workflows.models.object_field_selector import ObjectFieldSelector

# TODO update the JSON string below
json = "{}"
# create an instance of ObjectFieldSelector from a JSON string
object_field_selector_instance = ObjectFieldSelector.from_json(json)
# print the JSON string representation of the object
print(ObjectFieldSelector.to_json())

# convert the object into a dict
object_field_selector_dict = object_field_selector_instance.to_dict()
# create an instance of ObjectFieldSelector from a dict
object_field_selector_form_dict = object_field_selector.from_dict(object_field_selector_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


