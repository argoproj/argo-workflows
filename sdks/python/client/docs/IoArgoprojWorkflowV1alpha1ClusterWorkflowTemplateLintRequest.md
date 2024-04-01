# IoArgoprojWorkflowV1alpha1ClusterWorkflowTemplateLintRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**create_options** | [**CreateOptions**](CreateOptions.md) |  | [optional] 
**template** | [**IoArgoprojWorkflowV1alpha1ClusterWorkflowTemplate**](IoArgoprojWorkflowV1alpha1ClusterWorkflowTemplate.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cluster_workflow_template_lint_request import IoArgoprojWorkflowV1alpha1ClusterWorkflowTemplateLintRequest

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ClusterWorkflowTemplateLintRequest from a JSON string
io_argoproj_workflow_v1alpha1_cluster_workflow_template_lint_request_instance = IoArgoprojWorkflowV1alpha1ClusterWorkflowTemplateLintRequest.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ClusterWorkflowTemplateLintRequest.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_cluster_workflow_template_lint_request_dict = io_argoproj_workflow_v1alpha1_cluster_workflow_template_lint_request_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ClusterWorkflowTemplateLintRequest from a dict
io_argoproj_workflow_v1alpha1_cluster_workflow_template_lint_request_form_dict = io_argoproj_workflow_v1alpha1_cluster_workflow_template_lint_request.from_dict(io_argoproj_workflow_v1alpha1_cluster_workflow_template_lint_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


