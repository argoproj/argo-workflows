# IoArgoprojWorkflowV1alpha1HTTPHeader


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** |  | 
**value** | **str** |  | [optional] 
**value_from** | [**IoArgoprojWorkflowV1alpha1HTTPHeaderSource**](IoArgoprojWorkflowV1alpha1HTTPHeaderSource.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_http_header import IoArgoprojWorkflowV1alpha1HTTPHeader

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1HTTPHeader from a JSON string
io_argoproj_workflow_v1alpha1_http_header_instance = IoArgoprojWorkflowV1alpha1HTTPHeader.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1HTTPHeader.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_http_header_dict = io_argoproj_workflow_v1alpha1_http_header_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1HTTPHeader from a dict
io_argoproj_workflow_v1alpha1_http_header_form_dict = io_argoproj_workflow_v1alpha1_http_header.from_dict(io_argoproj_workflow_v1alpha1_http_header_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


