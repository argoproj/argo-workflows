# IoArgoprojWorkflowV1alpha1WorkflowEventBindingSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**event** | [**IoArgoprojWorkflowV1alpha1Event**](IoArgoprojWorkflowV1alpha1Event.md) |  | 
**submit** | [**IoArgoprojWorkflowV1alpha1Submit**](IoArgoprojWorkflowV1alpha1Submit.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_event_binding_spec import IoArgoprojWorkflowV1alpha1WorkflowEventBindingSpec

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowEventBindingSpec from a JSON string
io_argoproj_workflow_v1alpha1_workflow_event_binding_spec_instance = IoArgoprojWorkflowV1alpha1WorkflowEventBindingSpec.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowEventBindingSpec.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_event_binding_spec_dict = io_argoproj_workflow_v1alpha1_workflow_event_binding_spec_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowEventBindingSpec from a dict
io_argoproj_workflow_v1alpha1_workflow_event_binding_spec_form_dict = io_argoproj_workflow_v1alpha1_workflow_event_binding_spec.from_dict(io_argoproj_workflow_v1alpha1_workflow_event_binding_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


