# IoArgoprojWorkflowV1alpha1WorkflowWatchEvent


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**object** | [**IoArgoprojWorkflowV1alpha1Workflow**](IoArgoprojWorkflowV1alpha1Workflow.md) |  | [optional] 
**type** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_watch_event import IoArgoprojWorkflowV1alpha1WorkflowWatchEvent

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowWatchEvent from a JSON string
io_argoproj_workflow_v1alpha1_workflow_watch_event_instance = IoArgoprojWorkflowV1alpha1WorkflowWatchEvent.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowWatchEvent.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_watch_event_dict = io_argoproj_workflow_v1alpha1_workflow_watch_event_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowWatchEvent from a dict
io_argoproj_workflow_v1alpha1_workflow_watch_event_form_dict = io_argoproj_workflow_v1alpha1_workflow_watch_event.from_dict(io_argoproj_workflow_v1alpha1_workflow_watch_event_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


