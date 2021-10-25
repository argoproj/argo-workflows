# PreferredSchedulingTerm

An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**preference** | [**NodeSelectorTerm**](NodeSelectorTerm.md) |  | 
**weight** | **int** | Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100. | 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


