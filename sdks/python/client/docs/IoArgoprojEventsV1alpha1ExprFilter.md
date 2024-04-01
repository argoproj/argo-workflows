# IoArgoprojEventsV1alpha1ExprFilter


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**expr** | **str** | Expr refers to the expression that determines the outcome of the filter. | [optional] 
**fields** | [**List[IoArgoprojEventsV1alpha1PayloadField]**](IoArgoprojEventsV1alpha1PayloadField.md) | Fields refers to set of keys that refer to the paths within event payload. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_expr_filter import IoArgoprojEventsV1alpha1ExprFilter

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1ExprFilter from a JSON string
io_argoproj_events_v1alpha1_expr_filter_instance = IoArgoprojEventsV1alpha1ExprFilter.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1ExprFilter.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_expr_filter_dict = io_argoproj_events_v1alpha1_expr_filter_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1ExprFilter from a dict
io_argoproj_events_v1alpha1_expr_filter_form_dict = io_argoproj_events_v1alpha1_expr_filter.from_dict(io_argoproj_events_v1alpha1_expr_filter_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


