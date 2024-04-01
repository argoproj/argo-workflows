# IoArgoprojWorkflowV1alpha1WorkflowTaskSetStatus


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**nodes** | [**Dict[str, IoArgoprojWorkflowV1alpha1NodeResult]**](IoArgoprojWorkflowV1alpha1NodeResult.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_task_set_status import IoArgoprojWorkflowV1alpha1WorkflowTaskSetStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowTaskSetStatus from a JSON string
io_argoproj_workflow_v1alpha1_workflow_task_set_status_instance = IoArgoprojWorkflowV1alpha1WorkflowTaskSetStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowTaskSetStatus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_task_set_status_dict = io_argoproj_workflow_v1alpha1_workflow_task_set_status_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowTaskSetStatus from a dict
io_argoproj_workflow_v1alpha1_workflow_task_set_status_form_dict = io_argoproj_workflow_v1alpha1_workflow_task_set_status.from_dict(io_argoproj_workflow_v1alpha1_workflow_task_set_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


