# IoArgoprojWorkflowV1alpha1WorkflowSubmitRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**namespace** | **str** |  | [optional] 
**resource_kind** | **str** |  | [optional] 
**resource_name** | **str** |  | [optional] 
**submit_options** | [**IoArgoprojWorkflowV1alpha1SubmitOpts**](IoArgoprojWorkflowV1alpha1SubmitOpts.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_submit_request import IoArgoprojWorkflowV1alpha1WorkflowSubmitRequest

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowSubmitRequest from a JSON string
io_argoproj_workflow_v1alpha1_workflow_submit_request_instance = IoArgoprojWorkflowV1alpha1WorkflowSubmitRequest.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowSubmitRequest.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_submit_request_dict = io_argoproj_workflow_v1alpha1_workflow_submit_request_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowSubmitRequest from a dict
io_argoproj_workflow_v1alpha1_workflow_submit_request_form_dict = io_argoproj_workflow_v1alpha1_workflow_submit_request.from_dict(io_argoproj_workflow_v1alpha1_workflow_submit_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


