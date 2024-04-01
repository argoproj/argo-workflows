# IoArgoprojWorkflowV1alpha1WorkflowStep

WorkflowStep is a reference to a template to execute in a series of step

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**arguments** | [**IoArgoprojWorkflowV1alpha1Arguments**](IoArgoprojWorkflowV1alpha1Arguments.md) |  | [optional] 
**continue_on** | [**IoArgoprojWorkflowV1alpha1ContinueOn**](IoArgoprojWorkflowV1alpha1ContinueOn.md) |  | [optional] 
**hooks** | [**Dict[str, IoArgoprojWorkflowV1alpha1LifecycleHook]**](IoArgoprojWorkflowV1alpha1LifecycleHook.md) | Hooks holds the lifecycle hook which is invoked at lifecycle of step, irrespective of the success, failure, or error status of the primary step | [optional] 
**inline** | [**IoArgoprojWorkflowV1alpha1Template**](IoArgoprojWorkflowV1alpha1Template.md) |  | [optional] 
**name** | **str** | Name of the step | [optional] 
**on_exit** | **str** | OnExit is a template reference which is invoked at the end of the template, irrespective of the success, failure, or error of the primary template. DEPRECATED: Use Hooks[exit].Template instead. | [optional] 
**template** | **str** | Template is the name of the template to execute as the step | [optional] 
**template_ref** | [**IoArgoprojWorkflowV1alpha1TemplateRef**](IoArgoprojWorkflowV1alpha1TemplateRef.md) |  | [optional] 
**when** | **str** | When is an expression in which the step should conditionally execute | [optional] 
**with_items** | **List[object]** | WithItems expands a step into multiple parallel steps from the items in the list | [optional] 
**with_param** | **str** | WithParam expands a step into multiple parallel steps from the value in the parameter, which is expected to be a JSON list. | [optional] 
**with_sequence** | [**IoArgoprojWorkflowV1alpha1Sequence**](IoArgoprojWorkflowV1alpha1Sequence.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_workflow_step import IoArgoprojWorkflowV1alpha1WorkflowStep

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowStep from a JSON string
io_argoproj_workflow_v1alpha1_workflow_step_instance = IoArgoprojWorkflowV1alpha1WorkflowStep.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1WorkflowStep.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_workflow_step_dict = io_argoproj_workflow_v1alpha1_workflow_step_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1WorkflowStep from a dict
io_argoproj_workflow_v1alpha1_workflow_step_form_dict = io_argoproj_workflow_v1alpha1_workflow_step.from_dict(io_argoproj_workflow_v1alpha1_workflow_step_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


