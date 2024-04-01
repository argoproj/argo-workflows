# IoArgoprojWorkflowV1alpha1ContinueOn

ContinueOn defines if a workflow should continue even if a task or step fails/errors. It can be specified if the workflow should continue when the pod errors, fails or both.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**error** | **bool** |  | [optional] 
**failed** | **bool** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_continue_on import IoArgoprojWorkflowV1alpha1ContinueOn

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ContinueOn from a JSON string
io_argoproj_workflow_v1alpha1_continue_on_instance = IoArgoprojWorkflowV1alpha1ContinueOn.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ContinueOn.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_continue_on_dict = io_argoproj_workflow_v1alpha1_continue_on_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ContinueOn from a dict
io_argoproj_workflow_v1alpha1_continue_on_form_dict = io_argoproj_workflow_v1alpha1_continue_on.from_dict(io_argoproj_workflow_v1alpha1_continue_on_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


