# PodAffinityTerm

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**label_selector** | Option<[**crate::models::LabelSelector**](LabelSelector.md)> |  | [optional]
**namespace_selector** | Option<[**crate::models::LabelSelector**](LabelSelector.md)> |  | [optional]
**namespaces** | Option<**Vec<String>**> | namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector. null or empty namespaces list and null namespaceSelector means \"this pod's namespace\" | [optional]
**topology_key** | **String** | This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


