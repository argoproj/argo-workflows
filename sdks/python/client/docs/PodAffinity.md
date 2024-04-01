# PodAffinity

Pod affinity is a group of inter pod affinity scheduling rules.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**preferred_during_scheduling_ignored_during_execution** | [**List[WeightedPodAffinityTerm]**](WeightedPodAffinityTerm.md) | The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding \&quot;weight\&quot; to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred. | [optional] 
**required_during_scheduling_ignored_during_execution** | [**List[PodAffinityTerm]**](PodAffinityTerm.md) | If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied. | [optional] 

## Example

```python
from argo_workflows.models.pod_affinity import PodAffinity

# TODO update the JSON string below
json = "{}"
# create an instance of PodAffinity from a JSON string
pod_affinity_instance = PodAffinity.from_json(json)
# print the JSON string representation of the object
print(PodAffinity.to_json())

# convert the object into a dict
pod_affinity_dict = pod_affinity_instance.to_dict()
# create an instance of PodAffinity from a dict
pod_affinity_form_dict = pod_affinity.from_dict(pod_affinity_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


