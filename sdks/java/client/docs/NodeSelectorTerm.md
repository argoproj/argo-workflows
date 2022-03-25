

# NodeSelectorTerm

A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**matchExpressions** | [**List&lt;NodeSelectorRequirement&gt;**](NodeSelectorRequirement.md) | A list of node selector requirements by node&#39;s labels. |  [optional]
**matchFields** | [**List&lt;NodeSelectorRequirement&gt;**](NodeSelectorRequirement.md) | A list of node selector requirements by node&#39;s fields. |  [optional]



