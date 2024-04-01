# GroupVersionResource

+protobuf.options.(gogoproto.goproto_stringer)=false

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**group** | **str** |  | [optional] 
**resource** | **str** |  | [optional] 
**version** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.group_version_resource import GroupVersionResource

# TODO update the JSON string below
json = "{}"
# create an instance of GroupVersionResource from a JSON string
group_version_resource_instance = GroupVersionResource.from_json(json)
# print the JSON string representation of the object
print(GroupVersionResource.to_json())

# convert the object into a dict
group_version_resource_dict = group_version_resource_instance.to_dict()
# create an instance of GroupVersionResource from a dict
group_version_resource_form_dict = group_version_resource.from_dict(group_version_resource_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


