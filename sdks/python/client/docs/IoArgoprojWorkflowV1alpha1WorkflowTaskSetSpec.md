# IoArgoprojWorkflowV1alpha1WorkflowTaskSetSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**tasks** | [**Dict[str, IoArgoprojWorkflowV1alpha1Template]**](IoArgoprojWorkflowV1alpha1Template.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_task_set_spec import IoArgoprojWorkflowV1alpha1WorkflowTaskSetSpec

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowTaskSetSpec from a JSON string
io_argoproj_workflow_v1alpha1_workflow_task_set_spec_instance = IoArgoprojWorkflowV1alpha1WorkflowTaskSetSpec.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowTaskSetSpec.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_task_set_spec_dict = io_argoproj_workflow_v1alpha1_workflow_task_set_spec_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowTaskSetSpec from a dict
io_argoproj_workflow_v1alpha1_workflow_task_set_spec_form_dict = io_argoproj_workflow_v1alpha1_workflow_task_set_spec.from_dict(io_argoproj_workflow_v1alpha1_workflow_task_set_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


