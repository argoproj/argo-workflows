# PreferredSchedulingTerm

An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**preference** | [**NodeSelectorTerm**](NodeSelectorTerm.md) |  | 
**weight** | **int** | Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100. | 

## Example

```python
from argo_workflows.models.preferred_scheduling_term import PreferredSchedulingTerm

# TODO update the JSON string below
json = "{}"
# create an instance of PreferredSchedulingTerm from a JSON string
preferred_scheduling_term_instance = PreferredSchedulingTerm.from_json(json)
# print the JSON string representation of the object
print(PreferredSchedulingTerm.to_json())

# convert the object into a dict
preferred_scheduling_term_dict = preferred_scheduling_term_instance.to_dict()
# create an instance of PreferredSchedulingTerm from a dict
preferred_scheduling_term_form_dict = preferred_scheduling_term.from_dict(preferred_scheduling_term_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


