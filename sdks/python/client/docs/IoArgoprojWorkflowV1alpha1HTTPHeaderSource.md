# IoArgoprojWorkflowV1alpha1HTTPHeaderSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**secret_key_ref** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_http_header_source import IoArgoprojWorkflowV1alpha1HTTPHeaderSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1HTTPHeaderSource from a JSON string
io_argoproj_workflow_v1alpha1_http_header_source_instance = IoArgoprojWorkflowV1alpha1HTTPHeaderSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1HTTPHeaderSource.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_http_header_source_dict = io_argoproj_workflow_v1alpha1_http_header_source_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1HTTPHeaderSource from a dict
io_argoproj_workflow_v1alpha1_http_header_source_form_dict = io_argoproj_workflow_v1alpha1_http_header_source.from_dict(io_argoproj_workflow_v1alpha1_http_header_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


