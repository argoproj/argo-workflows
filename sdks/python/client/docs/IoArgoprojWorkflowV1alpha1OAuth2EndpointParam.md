# IoArgoprojWorkflowV1alpha1OAuth2EndpointParam

EndpointParam is for requesting optional fields that should be sent in the oauth request

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **str** | Name is the header name | 
**value** | **str** | Value is the literal value to use for the header | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_o_auth2_endpoint_param import IoArgoprojWorkflowV1alpha1OAuth2EndpointParam

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1OAuth2EndpointParam from a JSON string
io_argoproj_workflow_v1alpha1_o_auth2_endpoint_param_instance = IoArgoprojWorkflowV1alpha1OAuth2EndpointParam.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1OAuth2EndpointParam.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_o_auth2_endpoint_param_dict = io_argoproj_workflow_v1alpha1_o_auth2_endpoint_param_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1OAuth2EndpointParam from a dict
io_argoproj_workflow_v1alpha1_o_auth2_endpoint_param_form_dict = io_argoproj_workflow_v1alpha1_o_auth2_endpoint_param.from_dict(io_argoproj_workflow_v1alpha1_o_auth2_endpoint_param_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


