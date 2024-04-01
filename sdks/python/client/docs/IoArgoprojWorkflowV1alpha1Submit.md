# IoArgoprojWorkflowV1alpha1Submit


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**arguments** | [**IoArgoprojWorkflowV1alpha1Arguments**](IoArgoprojWorkflowV1alpha1Arguments.md) |  | [optional] 
**metadata** | [**ObjectMeta**](ObjectMeta.md) |  | [optional] 
**workflow_template_ref** | [**IoArgoprojWorkflowV1alpha1WorkflowTemplateRef**](IoArgoprojWorkflowV1alpha1WorkflowTemplateRef.md) |  | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_submit import IoArgoprojWorkflowV1alpha1Submit

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Submit from a JSON string
io_argoproj_workflow_v1alpha1_submit_instance = IoArgoprojWorkflowV1alpha1Submit.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Submit.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_submit_dict = io_argoproj_workflow_v1alpha1_submit_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Submit from a dict
io_argoproj_workflow_v1alpha1_submit_form_dict = io_argoproj_workflow_v1alpha1_submit.from_dict(io_argoproj_workflow_v1alpha1_submit_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


