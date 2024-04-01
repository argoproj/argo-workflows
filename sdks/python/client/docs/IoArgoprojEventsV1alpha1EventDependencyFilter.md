# IoArgoprojEventsV1alpha1EventDependencyFilter

EventDependencyFilter defines filters and constraints for a io.argoproj.workflow.v1alpha1.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**context** | [**IoArgoprojEventsV1alpha1EventContext**](IoArgoprojEventsV1alpha1EventContext.md) |  | [optional] 
**data** | [**List[IoArgoprojEventsV1alpha1DataFilter]**](IoArgoprojEventsV1alpha1DataFilter.md) |  | [optional] 
**data_logical_operator** | **str** | DataLogicalOperator defines how multiple Data filters (if defined) are evaluated together. Available values: and (&amp;&amp;), or (||) Is optional and if left blank treated as and (&amp;&amp;). | [optional] 
**expr_logical_operator** | **str** | ExprLogicalOperator defines how multiple Exprs filters (if defined) are evaluated together. Available values: and (&amp;&amp;), or (||) Is optional and if left blank treated as and (&amp;&amp;). | [optional] 
**exprs** | [**List[IoArgoprojEventsV1alpha1ExprFilter]**](IoArgoprojEventsV1alpha1ExprFilter.md) | Exprs contains the list of expressions evaluated against the event payload. | [optional] 
**script** | **str** | Script refers to a Lua script evaluated to determine the validity of an io.argoproj.workflow.v1alpha1. | [optional] 
**time** | [**IoArgoprojEventsV1alpha1TimeFilter**](IoArgoprojEventsV1alpha1TimeFilter.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_event_dependency_filter import IoArgoprojEventsV1alpha1EventDependencyFilter

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1EventDependencyFilter from a JSON string
io_argoproj_events_v1alpha1_event_dependency_filter_instance = IoArgoprojEventsV1alpha1EventDependencyFilter.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1EventDependencyFilter.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_event_dependency_filter_dict = io_argoproj_events_v1alpha1_event_dependency_filter_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1EventDependencyFilter from a dict
io_argoproj_events_v1alpha1_event_dependency_filter_form_dict = io_argoproj_events_v1alpha1_event_dependency_filter.from_dict(io_argoproj_events_v1alpha1_event_dependency_filter_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


