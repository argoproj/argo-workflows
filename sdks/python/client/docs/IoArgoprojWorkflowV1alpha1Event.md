# IoArgoprojWorkflowV1alpha1Event


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**selector** | **str** | Selector (https://github.com/expr-lang/expr) that we must must match the io.argoproj.workflow.v1alpha1. E.g. &#x60;payload.message &#x3D;&#x3D; \&quot;test\&quot;&#x60; | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_event import IoArgoprojWorkflowV1alpha1Event

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Event from a JSON string
io_argoproj_workflow_v1alpha1_event_instance = IoArgoprojWorkflowV1alpha1Event.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Event.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_event_dict = io_argoproj_workflow_v1alpha1_event_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Event from a dict
io_argoproj_workflow_v1alpha1_event_form_dict = io_argoproj_workflow_v1alpha1_event.from_dict(io_argoproj_workflow_v1alpha1_event_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


