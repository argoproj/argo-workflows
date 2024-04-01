# IoArgoprojWorkflowV1alpha1WorkflowTemplateRef

WorkflowTemplateRef is a reference to a WorkflowTemplate resource.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cluster_scope** | **bool** | ClusterScope indicates the referred template is cluster scoped (i.e. a ClusterWorkflowTemplate). | [optional] 
**name** | **str** | Name is the resource name of the workflow template. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_template_ref import IoArgoprojWorkflowV1alpha1WorkflowTemplateRef

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowTemplateRef from a JSON string
io_argoproj_workflow_v1alpha1_workflow_template_ref_instance = IoArgoprojWorkflowV1alpha1WorkflowTemplateRef.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowTemplateRef.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_template_ref_dict = io_argoproj_workflow_v1alpha1_workflow_template_ref_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowTemplateRef from a dict
io_argoproj_workflow_v1alpha1_workflow_template_ref_form_dict = io_argoproj_workflow_v1alpha1_workflow_template_ref.from_dict(io_argoproj_workflow_v1alpha1_workflow_template_ref_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


