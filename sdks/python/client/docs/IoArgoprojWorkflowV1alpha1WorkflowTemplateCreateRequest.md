# IoArgoprojWorkflowV1alpha1WorkflowTemplateCreateRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**create_options** | [**CreateOptions**](CreateOptions.md) |  | [optional] 
**namespace** | **str** |  | [optional] 
**template** | [**IoArgoprojWorkflowV1alpha1WorkflowTemplate**](IoArgoprojWorkflowV1alpha1WorkflowTemplate.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_template_create_request import IoArgoprojWorkflowV1alpha1WorkflowTemplateCreateRequest

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowTemplateCreateRequest from a JSON string
io_argoproj_workflow_v1alpha1_workflow_template_create_request_instance = IoArgoprojWorkflowV1alpha1WorkflowTemplateCreateRequest.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowTemplateCreateRequest.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_template_create_request_dict = io_argoproj_workflow_v1alpha1_workflow_template_create_request_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowTemplateCreateRequest from a dict
io_argoproj_workflow_v1alpha1_workflow_template_create_request_form_dict = io_argoproj_workflow_v1alpha1_workflow_template_create_request.from_dict(io_argoproj_workflow_v1alpha1_workflow_template_create_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


