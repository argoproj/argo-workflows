# IoArgoprojWorkflowV1alpha1Counter

Counter is a Counter prometheus metric

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**value** | **str** | Value is the value of the metric | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_counter import IoArgoprojWorkflowV1alpha1Counter

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Counter from a JSON string
io_argoproj_workflow_v1alpha1_counter_instance = IoArgoprojWorkflowV1alpha1Counter.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Counter.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_counter_dict = io_argoproj_workflow_v1alpha1_counter_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Counter from a dict
io_argoproj_workflow_v1alpha1_counter_form_dict = io_argoproj_workflow_v1alpha1_counter.from_dict(io_argoproj_workflow_v1alpha1_counter_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


