# IoArgoprojWorkflowV1alpha1ResubmitArchivedWorkflowRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**memoized** | **bool** |  | [optional] 
**name** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 
**parameters** | **List[str]** |  | [optional] 
**uid** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_resubmit_archived_workflow_request import IoArgoprojWorkflowV1alpha1ResubmitArchivedWorkflowRequest

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ResubmitArchivedWorkflowRequest from a JSON string
io_argoproj_workflow_v1alpha1_resubmit_archived_workflow_request_instance = IoArgoprojWorkflowV1alpha1ResubmitArchivedWorkflowRequest.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ResubmitArchivedWorkflowRequest.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_resubmit_archived_workflow_request_dict = io_argoproj_workflow_v1alpha1_resubmit_archived_workflow_request_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ResubmitArchivedWorkflowRequest from a dict
io_argoproj_workflow_v1alpha1_resubmit_archived_workflow_request_form_dict = io_argoproj_workflow_v1alpha1_resubmit_archived_workflow_request.from_dict(io_argoproj_workflow_v1alpha1_resubmit_archived_workflow_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


