# IoArgoprojWorkflowV1alpha1GetUserInfoResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**email** | **str** |  | [optional] 
**email_verified** | **bool** |  | [optional] 
**groups** | **List[str]** |  | [optional] 
**issuer** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**service_account_name** | **str** |  | [optional] 
**service_account_namespace** | **str** |  | [optional] 
**subject** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_get_user_info_response import IoArgoprojWorkflowV1alpha1GetUserInfoResponse

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1GetUserInfoResponse from a JSON string
io_argoproj_workflow_v1alpha1_get_user_info_response_instance = IoArgoprojWorkflowV1alpha1GetUserInfoResponse.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1GetUserInfoResponse.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_get_user_info_response_dict = io_argoproj_workflow_v1alpha1_get_user_info_response_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1GetUserInfoResponse from a dict
io_argoproj_workflow_v1alpha1_get_user_info_response_form_dict = io_argoproj_workflow_v1alpha1_get_user_info_response.from_dict(io_argoproj_workflow_v1alpha1_get_user_info_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


