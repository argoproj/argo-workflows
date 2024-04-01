# IoArgoprojWorkflowV1alpha1WorkflowSetRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**message** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 
**node_field_selector** | **str** |  | [optional] 
**output_parameters** | **str** |  | [optional] 
**phase** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_set_request import IoArgoprojWorkflowV1alpha1WorkflowSetRequest

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowSetRequest from a JSON string
io_argoproj_workflow_v1alpha1_workflow_set_request_instance = IoArgoprojWorkflowV1alpha1WorkflowSetRequest.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowSetRequest.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_set_request_dict = io_argoproj_workflow_v1alpha1_workflow_set_request_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowSetRequest from a dict
io_argoproj_workflow_v1alpha1_workflow_set_request_form_dict = io_argoproj_workflow_v1alpha1_workflow_set_request.from_dict(io_argoproj_workflow_v1alpha1_workflow_set_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


