# WeightedPodAffinityTerm

The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**pod_affinity_term** | [**PodAffinityTerm**](PodAffinityTerm.md) |  | 
**weight** | **int** | weight associated with matching the corresponding podAffinityTerm, in the range 1-100. | 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


