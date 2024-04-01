# IoArgoprojWorkflowV1alpha1HTTPAuth


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**basic_auth** | [**IoArgoprojWorkflowV1alpha1BasicAuth**](IoArgoprojWorkflowV1alpha1BasicAuth.md) |  | [optional] 
**client_cert** | [**IoArgoprojWorkflowV1alpha1ClientCertAuth**](IoArgoprojWorkflowV1alpha1ClientCertAuth.md) |  | [optional] 
**oauth2** | [**IoArgoprojWorkflowV1alpha1OAuth2Auth**](IoArgoprojWorkflowV1alpha1OAuth2Auth.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_http_auth import IoArgoprojWorkflowV1alpha1HTTPAuth

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1HTTPAuth from a JSON string
io_argoproj_workflow_v1alpha1_http_auth_instance = IoArgoprojWorkflowV1alpha1HTTPAuth.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1HTTPAuth.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_http_auth_dict = io_argoproj_workflow_v1alpha1_http_auth_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1HTTPAuth from a dict
io_argoproj_workflow_v1alpha1_http_auth_form_dict = io_argoproj_workflow_v1alpha1_http_auth.from_dict(io_argoproj_workflow_v1alpha1_http_auth_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


