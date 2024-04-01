# IoArgoprojWorkflowV1alpha1ClientCertAuth

ClientCertAuth holds necessary information for client authentication via certificates

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**client_cert_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**client_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_client_cert_auth import IoArgoprojWorkflowV1alpha1ClientCertAuth

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ClientCertAuth from a JSON string
io_argoproj_workflow_v1alpha1_client_cert_auth_instance = IoArgoprojWorkflowV1alpha1ClientCertAuth.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ClientCertAuth.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_client_cert_auth_dict = io_argoproj_workflow_v1alpha1_client_cert_auth_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ClientCertAuth from a dict
io_argoproj_workflow_v1alpha1_client_cert_auth_form_dict = io_argoproj_workflow_v1alpha1_client_cert_auth.from_dict(io_argoproj_workflow_v1alpha1_client_cert_auth_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


