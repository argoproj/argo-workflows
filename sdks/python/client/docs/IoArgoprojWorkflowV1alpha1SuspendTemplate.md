# IoArgoprojWorkflowV1alpha1SuspendTemplate

SuspendTemplate is a template subtype to suspend a workflow at a predetermined point in time

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**duration** | **str** | Duration is the seconds to wait before automatically resuming a template. Must be a string. Default unit is seconds. Could also be a Duration, e.g.: \&quot;2m\&quot;, \&quot;6h\&quot; | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_suspend_template import IoArgoprojWorkflowV1alpha1SuspendTemplate

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1SuspendTemplate from a JSON string
io_argoproj_workflow_v1alpha1_suspend_template_instance = IoArgoprojWorkflowV1alpha1SuspendTemplate.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1SuspendTemplate.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_suspend_template_dict = io_argoproj_workflow_v1alpha1_suspend_template_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1SuspendTemplate from a dict
io_argoproj_workflow_v1alpha1_suspend_template_form_dict = io_argoproj_workflow_v1alpha1_suspend_template.from_dict(io_argoproj_workflow_v1alpha1_suspend_template_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


