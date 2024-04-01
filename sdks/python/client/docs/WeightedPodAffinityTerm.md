# WeightedPodAffinityTerm

The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**pod_affinity_term** | [**PodAffinityTerm**](PodAffinityTerm.md) |  | 
**weight** | **int** | weight associated with matching the corresponding podAffinityTerm, in the range 1-100. | 

## Example

```python
from argo_workflows.models.weighted_pod_affinity_term import WeightedPodAffinityTerm

# TODO update the JSON string below
json = "{}"
# create an instance of WeightedPodAffinityTerm from a JSON string
weighted_pod_affinity_term_instance = WeightedPodAffinityTerm.from_json(json)
# print the JSON string representation of the object
print(WeightedPodAffinityTerm.to_json())

# convert the object into a dict
weighted_pod_affinity_term_dict = weighted_pod_affinity_term_instance.to_dict()
# create an instance of WeightedPodAffinityTerm from a dict
weighted_pod_affinity_term_form_dict = weighted_pod_affinity_term.from_dict(weighted_pod_affinity_term_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


