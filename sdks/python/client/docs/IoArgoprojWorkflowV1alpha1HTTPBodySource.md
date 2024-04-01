# IoArgoprojWorkflowV1alpha1HTTPBodySource

HTTPBodySource contains the source of the HTTP body.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**bytes** | **bytearray** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_http_body_source import IoArgoprojWorkflowV1alpha1HTTPBodySource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1HTTPBodySource from a JSON string
io_argoproj_workflow_v1alpha1_http_body_source_instance = IoArgoprojWorkflowV1alpha1HTTPBodySource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1HTTPBodySource.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_http_body_source_dict = io_argoproj_workflow_v1alpha1_http_body_source_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1HTTPBodySource from a dict
io_argoproj_workflow_v1alpha1_http_body_source_form_dict = io_argoproj_workflow_v1alpha1_http_body_source.from_dict(io_argoproj_workflow_v1alpha1_http_body_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


