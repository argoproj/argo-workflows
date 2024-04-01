# NodeSelector

A node selector represents the union of the results of one or more label queries over a set of nodes; that is, it represents the OR of the selectors represented by the node selector terms.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**node_selector_terms** | [**List[NodeSelectorTerm]**](NodeSelectorTerm.md) | Required. A list of node selector terms. The terms are ORed. | 

## Example

```python
from argo_workflows.models.node_selector import NodeSelector

# TODO update the JSON string below
json = "{}"
# create an instance of NodeSelector from a JSON string
node_selector_instance = NodeSelector.from_json(json)
# print the JSON string representation of the object
print(NodeSelector.to_json())

# convert the object into a dict
node_selector_dict = node_selector_instance.to_dict()
# create an instance of NodeSelector from a dict
node_selector_form_dict = node_selector.from_dict(node_selector_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


