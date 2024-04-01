# NodeSelectorRequirement

A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **str** | The label key that the selector applies to. | 
**operator** | **str** | Represents a key&#39;s relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.  Possible enum values:  - &#x60;\&quot;DoesNotExist\&quot;&#x60;  - &#x60;\&quot;Exists\&quot;&#x60;  - &#x60;\&quot;Gt\&quot;&#x60;  - &#x60;\&quot;In\&quot;&#x60;  - &#x60;\&quot;Lt\&quot;&#x60;  - &#x60;\&quot;NotIn\&quot;&#x60; | 
**values** | **List[str]** | An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch. | [optional] 

## Example

```python
from argo_workflows.models.node_selector_requirement import NodeSelectorRequirement

# TODO update the JSON string below
json = "{}"
# create an instance of NodeSelectorRequirement from a JSON string
node_selector_requirement_instance = NodeSelectorRequirement.from_json(json)
# print the JSON string representation of the object
print(NodeSelectorRequirement.to_json())

# convert the object into a dict
node_selector_requirement_dict = node_selector_requirement_instance.to_dict()
# create an instance of NodeSelectorRequirement from a dict
node_selector_requirement_form_dict = node_selector_requirement.from_dict(node_selector_requirement_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


