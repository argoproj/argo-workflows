# IoArgoprojWorkflowV1alpha1DAGTask

DAGTask represents a node in the graph during DAG execution

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name is the name of the target | 
**arguments** | [**IoArgoprojWorkflowV1alpha1Arguments**](IoArgoprojWorkflowV1alpha1Arguments.md) |  | [optional] 
**continue_on** | [**IoArgoprojWorkflowV1alpha1ContinueOn**](IoArgoprojWorkflowV1alpha1ContinueOn.md) |  | [optional] 
**dependencies** | **[str]** | Dependencies are name of other targets which this depends on | [optional] 
**depends** | **str** | Depends are name of other targets which this depends on | [optional] 
**description** | **str** | Description of the DAGTask | [optional] 
**hooks** | [**{str: (IoArgoprojWorkflowV1alpha1LifecycleHook,)}**](IoArgoprojWorkflowV1alpha1LifecycleHook.md) | Hooks hold the lifecycle hook which is invoked at lifecycle of task, irrespective of the success, failure, or error status of the primary task | [optional] 
**inline** | [**IoArgoprojWorkflowV1alpha1Template**](IoArgoprojWorkflowV1alpha1Template.md) |  | [optional] 
**on_exit** | **str** | OnExit is a template reference which is invoked at the end of the template, irrespective of the success, failure, or error of the primary template. DEPRECATED: Use Hooks[exit].Template instead. | [optional] 
**template** | **str** | Name of template to execute | [optional] 
**template_ref** | [**IoArgoprojWorkflowV1alpha1TemplateRef**](IoArgoprojWorkflowV1alpha1TemplateRef.md) |  | [optional] 
**when** | **str** | When is an expression in which the task should conditionally execute | [optional] 
**with_items** | **[dict]** | WithItems expands a task into multiple parallel tasks from the items in the list | [optional] 
**with_param** | **str** | WithParam expands a task into multiple parallel tasks from the value in the parameter, which is expected to be a JSON list. | [optional] 
**with_sequence** | [**IoArgoprojWorkflowV1alpha1Sequence**](IoArgoprojWorkflowV1alpha1Sequence.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


