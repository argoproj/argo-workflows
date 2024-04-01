# IoArgoprojWorkflowV1alpha1WorkflowStopRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**message** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 
**node_field_selector** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_stop_request import IoArgoprojWorkflowV1alpha1WorkflowStopRequest

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowStopRequest from a JSON string
io_argoproj_workflow_v1alpha1_workflow_stop_request_instance = IoArgoprojWorkflowV1alpha1WorkflowStopRequest.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowStopRequest.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_stop_request_dict = io_argoproj_workflow_v1alpha1_workflow_stop_request_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowStopRequest from a dict
io_argoproj_workflow_v1alpha1_workflow_stop_request_form_dict = io_argoproj_workflow_v1alpha1_workflow_stop_request.from_dict(io_argoproj_workflow_v1alpha1_workflow_stop_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


