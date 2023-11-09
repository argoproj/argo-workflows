# NodeSelector

A node selector represents the union of the results of one or more label queries over a set of nodes; that is, it represents the OR of the selectors represented by the node selector terms.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**node_selector_terms** | [**[NodeSelectorTerm]**](NodeSelectorTerm.md) | Required. A list of node selector terms. The terms are ORed. | 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


