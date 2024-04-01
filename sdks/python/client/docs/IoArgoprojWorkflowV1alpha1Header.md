# IoArgoprojWorkflowV1alpha1Header

Header indicate a key-value request header to be used when fetching artifacts over HTTP

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name is the header name | 
**value** | **str** | Value is the literal value to use for the header | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_header import IoArgoprojWorkflowV1alpha1Header

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Header from a JSON string
io_argoproj_workflow_v1alpha1_header_instance = IoArgoprojWorkflowV1alpha1Header.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Header.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_header_dict = io_argoproj_workflow_v1alpha1_header_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Header from a dict
io_argoproj_workflow_v1alpha1_header_form_dict = io_argoproj_workflow_v1alpha1_header.from_dict(io_argoproj_workflow_v1alpha1_header_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


