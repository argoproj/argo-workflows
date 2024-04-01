# IoArgoprojWorkflowV1alpha1WorkflowCreateRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**create_options** | [**CreateOptions**](CreateOptions.md) |  | [optional] 
**instance_id** | **str** | This field is no longer used. | [optional] 
**namespace** | **str** |  | [optional] 
**server_dry_run** | **bool** |  | [optional] 
**workflow** | [**IoArgoprojWorkflowV1alpha1Workflow**](IoArgoprojWorkflowV1alpha1Workflow.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_create_request import IoArgoprojWorkflowV1alpha1WorkflowCreateRequest

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowCreateRequest from a JSON string
io_argoproj_workflow_v1alpha1_workflow_create_request_instance = IoArgoprojWorkflowV1alpha1WorkflowCreateRequest.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowCreateRequest.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_create_request_dict = io_argoproj_workflow_v1alpha1_workflow_create_request_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowCreateRequest from a dict
io_argoproj_workflow_v1alpha1_workflow_create_request_form_dict = io_argoproj_workflow_v1alpha1_workflow_create_request.from_dict(io_argoproj_workflow_v1alpha1_workflow_create_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


