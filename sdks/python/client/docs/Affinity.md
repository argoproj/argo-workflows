# Affinity

Affinity is a group of affinity scheduling rules.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**node_affinity** | [**NodeAffinity**](NodeAffinity.md) |  | [optional] 
**pod_affinity** | [**PodAffinity**](PodAffinity.md) |  | [optional] 
**pod_anti_affinity** | [**PodAntiAffinity**](PodAntiAffinity.md) |  | [optional] 

## Example

```python
from argo_workflows.models.affinity import Affinity

# TODO update the JSON string below
json = "{}"
# create an instance of Affinity from a JSON string
affinity_instance = Affinity.from_json(json)
# print the JSON string representation of the object
print(Affinity.to_json())

# convert the object into a dict
affinity_dict = affinity_instance.to_dict()
# create an instance of Affinity from a dict
affinity_form_dict = affinity.from_dict(affinity_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


