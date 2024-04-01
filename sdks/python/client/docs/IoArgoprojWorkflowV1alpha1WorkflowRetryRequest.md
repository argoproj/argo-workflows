# IoArgoprojWorkflowV1alpha1WorkflowRetryRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 
**node_field_selector** | **str** |  | [optional] 
**parameters** | **List[str]** |  | [optional] 
**restart_successful** | **bool** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_retry_request import IoArgoprojWorkflowV1alpha1WorkflowRetryRequest

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowRetryRequest from a JSON string
io_argoproj_workflow_v1alpha1_workflow_retry_request_instance = IoArgoprojWorkflowV1alpha1WorkflowRetryRequest.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowRetryRequest.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_retry_request_dict = io_argoproj_workflow_v1alpha1_workflow_retry_request_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowRetryRequest from a dict
io_argoproj_workflow_v1alpha1_workflow_retry_request_form_dict = io_argoproj_workflow_v1alpha1_workflow_retry_request.from_dict(io_argoproj_workflow_v1alpha1_workflow_retry_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


