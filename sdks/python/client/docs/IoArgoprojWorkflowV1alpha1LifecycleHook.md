# IoArgoprojWorkflowV1alpha1LifecycleHook


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**arguments** | [**IoArgoprojWorkflowV1alpha1Arguments**](IoArgoprojWorkflowV1alpha1Arguments.md) |  | [optional] 
**expression** | **str** | Expression is a condition expression for when a node will be retried. If it evaluates to false, the node will not be retried and the retry strategy will be ignored | [optional] 
**template** | **str** | Template is the name of the template to execute by the hook | [optional] 
**template_ref** | [**IoArgoprojWorkflowV1alpha1TemplateRef**](IoArgoprojWorkflowV1alpha1TemplateRef.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_lifecycle_hook import IoArgoprojWorkflowV1alpha1LifecycleHook

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1LifecycleHook from a JSON string
io_argoproj_workflow_v1alpha1_lifecycle_hook_instance = IoArgoprojWorkflowV1alpha1LifecycleHook.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1LifecycleHook.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_lifecycle_hook_dict = io_argoproj_workflow_v1alpha1_lifecycle_hook_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1LifecycleHook from a dict
io_argoproj_workflow_v1alpha1_lifecycle_hook_form_dict = io_argoproj_workflow_v1alpha1_lifecycle_hook.from_dict(io_argoproj_workflow_v1alpha1_lifecycle_hook_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


