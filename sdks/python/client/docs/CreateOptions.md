# CreateOptions

CreateOptions may be provided when creating an API object.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**dry_run** | **List[str]** |  | [optional] 
**field_manager** | **str** |  | [optional] 
**field_validation** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.create_options import CreateOptions

# TODO update the JSON string below
json = "{}"
# create an instance of CreateOptions from a JSON string
create_options_instance = CreateOptions.from_json(json)
# print the JSON string representation of the object
print(CreateOptions.to_json())

# convert the object into a dict
create_options_dict = create_options_instance.to_dict()
# create an instance of CreateOptions from a dict
create_options_form_dict = create_options.from_dict(create_options_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


