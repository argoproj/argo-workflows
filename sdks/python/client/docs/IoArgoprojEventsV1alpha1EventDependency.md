# IoArgoprojEventsV1alpha1EventDependency


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**event_name** | **str** |  | [optional] 
**event_source_name** | **str** |  | [optional] 
**filters** | [**IoArgoprojEventsV1alpha1EventDependencyFilter**](IoArgoprojEventsV1alpha1EventDependencyFilter.md) |  | [optional] 
**filters_logical_operator** | **str** | FiltersLogicalOperator defines how different filters are evaluated together. Available values: and (&amp;&amp;), or (||) Is optional and if left blank treated as and (&amp;&amp;). | [optional] 
**name** | **str** |  | [optional] 
**transform** | [**IoArgoprojEventsV1alpha1EventDependencyTransformer**](IoArgoprojEventsV1alpha1EventDependencyTransformer.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_event_dependency import IoArgoprojEventsV1alpha1EventDependency

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1EventDependency from a JSON string
io_argoproj_events_v1alpha1_event_dependency_instance = IoArgoprojEventsV1alpha1EventDependency.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1EventDependency.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_event_dependency_dict = io_argoproj_events_v1alpha1_event_dependency_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1EventDependency from a dict
io_argoproj_events_v1alpha1_event_dependency_form_dict = io_argoproj_events_v1alpha1_event_dependency.from_dict(io_argoproj_events_v1alpha1_event_dependency_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


