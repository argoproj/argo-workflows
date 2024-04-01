# IoArgoprojWorkflowV1alpha1WorkflowLintRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**namespace** | **str** |  | [optional] 
**workflow** | [**IoArgoprojWorkflowV1alpha1Workflow**](IoArgoprojWorkflowV1alpha1Workflow.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_lint_request import IoArgoprojWorkflowV1alpha1WorkflowLintRequest

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowLintRequest from a JSON string
io_argoproj_workflow_v1alpha1_workflow_lint_request_instance = IoArgoprojWorkflowV1alpha1WorkflowLintRequest.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowLintRequest.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_lint_request_dict = io_argoproj_workflow_v1alpha1_workflow_lint_request_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowLintRequest from a dict
io_argoproj_workflow_v1alpha1_workflow_lint_request_form_dict = io_argoproj_workflow_v1alpha1_workflow_lint_request.from_dict(io_argoproj_workflow_v1alpha1_workflow_lint_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


