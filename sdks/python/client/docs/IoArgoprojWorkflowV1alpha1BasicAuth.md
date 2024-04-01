# IoArgoprojWorkflowV1alpha1BasicAuth

BasicAuth describes the secret selectors required for basic authentication

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**password_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**username_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_basic_auth import IoArgoprojWorkflowV1alpha1BasicAuth

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1BasicAuth from a JSON string
io_argoproj_workflow_v1alpha1_basic_auth_instance = IoArgoprojWorkflowV1alpha1BasicAuth.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1BasicAuth.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_basic_auth_dict = io_argoproj_workflow_v1alpha1_basic_auth_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1BasicAuth from a dict
io_argoproj_workflow_v1alpha1_basic_auth_form_dict = io_argoproj_workflow_v1alpha1_basic_auth.from_dict(io_argoproj_workflow_v1alpha1_basic_auth_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


