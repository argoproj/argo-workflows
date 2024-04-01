# NodeSelectorTerm

A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**match_expressions** | [**List[NodeSelectorRequirement]**](NodeSelectorRequirement.md) | A list of node selector requirements by node&#39;s labels. | [optional] 
**match_fields** | [**List[NodeSelectorRequirement]**](NodeSelectorRequirement.md) | A list of node selector requirements by node&#39;s fields. | [optional] 

## Example

```python
from argo_workflows.models.node_selector_term import NodeSelectorTerm

# TODO update the JSON string below
json = "{}"
# create an instance of NodeSelectorTerm from a JSON string
node_selector_term_instance = NodeSelectorTerm.from_json(json)
# print the JSON string representation of the object
print(NodeSelectorTerm.to_json())

# convert the object into a dict
node_selector_term_dict = node_selector_term_instance.to_dict()
# create an instance of NodeSelectorTerm from a dict
node_selector_term_form_dict = node_selector_term.from_dict(node_selector_term_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


