# LabelSelectorRequirement

A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **str** | key is the label key that the selector applies to. | 
**operator** | **str** | operator represents a key&#39;s relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist. | 
**values** | **List[str]** | values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch. | [optional] 

## Example

```python
from argo_workflows.models.label_selector_requirement import LabelSelectorRequirement

# TODO update the JSON string below
json = "{}"
# create an instance of LabelSelectorRequirement from a JSON string
label_selector_requirement_instance = LabelSelectorRequirement.from_json(json)
# print the JSON string representation of the object
print(LabelSelectorRequirement.to_json())

# convert the object into a dict
label_selector_requirement_dict = label_selector_requirement_instance.to_dict()
# create an instance of LabelSelectorRequirement from a dict
label_selector_requirement_form_dict = label_selector_requirement.from_dict(label_selector_requirement_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


