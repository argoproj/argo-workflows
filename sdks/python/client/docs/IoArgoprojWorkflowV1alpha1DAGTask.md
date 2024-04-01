# IoArgoprojWorkflowV1alpha1DAGTask

DAGTask represents a node in the graph during DAG execution

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**arguments** | [**IoArgoprojWorkflowV1alpha1Arguments**](IoArgoprojWorkflowV1alpha1Arguments.md) |  | [optional] 
**continue_on** | [**IoArgoprojWorkflowV1alpha1ContinueOn**](IoArgoprojWorkflowV1alpha1ContinueOn.md) |  | [optional] 
**dependencies** | **List[str]** | Dependencies are name of other targets which this depends on | [optional] 
**depends** | **str** | Depends are name of other targets which this depends on | [optional] 
**hooks** | [**Dict[str, IoArgoprojWorkflowV1alpha1LifecycleHook]**](IoArgoprojWorkflowV1alpha1LifecycleHook.md) | Hooks hold the lifecycle hook which is invoked at lifecycle of task, irrespective of the success, failure, or error status of the primary task | [optional] 
**inline** | [**IoArgoprojWorkflowV1alpha1Template**](IoArgoprojWorkflowV1alpha1Template.md) |  | [optional] 
**name** | **str** | Name is the name of the target | 
**on_exit** | **str** | OnExit is a template reference which is invoked at the end of the template, irrespective of the success, failure, or error of the primary template. DEPRECATED: Use Hooks[exit].Template instead. | [optional] 
**template** | **str** | Name of template to execute | [optional] 
**template_ref** | [**IoArgoprojWorkflowV1alpha1TemplateRef**](IoArgoprojWorkflowV1alpha1TemplateRef.md) |  | [optional] 
**when** | **str** | When is an expression in which the task should conditionally execute | [optional] 
**with_items** | **List[object]** | WithItems expands a task into multiple parallel tasks from the items in the list | [optional] 
**with_param** | **str** | WithParam expands a task into multiple parallel tasks from the value in the parameter, which is expected to be a JSON list. | [optional] 
**with_sequence** | [**IoArgoprojWorkflowV1alpha1Sequence**](IoArgoprojWorkflowV1alpha1Sequence.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_dag_task import IoArgoprojWorkflowV1alpha1DAGTask

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1DAGTask from a JSON string
io_argoproj_workflow_v1alpha1_dag_task_instance = IoArgoprojWorkflowV1alpha1DAGTask.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1DAGTask.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_dag_task_dict = io_argoproj_workflow_v1alpha1_dag_task_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1DAGTask from a dict
io_argoproj_workflow_v1alpha1_dag_task_form_dict = io_argoproj_workflow_v1alpha1_dag_task.from_dict(io_argoproj_workflow_v1alpha1_dag_task_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


