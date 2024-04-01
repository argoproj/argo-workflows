# IoArgoprojWorkflowV1alpha1OAuth2Auth

OAuth2Auth holds all information for client authentication via OAuth2 tokens

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**client_id_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**client_secret_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**endpoint_params** | [**List[IoArgoprojWorkflowV1alpha1OAuth2EndpointParam]**](IoArgoprojWorkflowV1alpha1OAuth2EndpointParam.md) |  | [optional] 
**scopes** | **List[str]** |  | [optional] 
**token_url_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_o_auth2_auth import IoArgoprojWorkflowV1alpha1OAuth2Auth

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1OAuth2Auth from a JSON string
io_argoproj_workflow_v1alpha1_o_auth2_auth_instance = IoArgoprojWorkflowV1alpha1OAuth2Auth.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1OAuth2Auth.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_o_auth2_auth_dict = io_argoproj_workflow_v1alpha1_o_auth2_auth_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1OAuth2Auth from a dict
io_argoproj_workflow_v1alpha1_o_auth2_auth_form_dict = io_argoproj_workflow_v1alpha1_o_auth2_auth.from_dict(io_argoproj_workflow_v1alpha1_o_auth2_auth_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


