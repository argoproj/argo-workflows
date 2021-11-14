

# PreferredSchedulingTerm

An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**preference** | [**NodeSelectorTerm**](NodeSelectorTerm.md) |  | 
**weight** | **Integer** | Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100. | 



