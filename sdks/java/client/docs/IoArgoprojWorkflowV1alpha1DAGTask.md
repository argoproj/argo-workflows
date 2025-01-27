

# IoArgoprojWorkflowV1alpha1DAGTask

DAGTask represents a node in the graph during DAG execution

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**arguments** | [**IoArgoprojWorkflowV1alpha1Arguments**](IoArgoprojWorkflowV1alpha1Arguments.md) |  |  [optional]
**continueOn** | [**IoArgoprojWorkflowV1alpha1ContinueOn**](IoArgoprojWorkflowV1alpha1ContinueOn.md) |  |  [optional]
**dependencies** | **List&lt;String&gt;** | Dependencies are name of other targets which this depends on |  [optional]
**depends** | **String** | Depends are name of other targets which this depends on |  [optional]
**hooks** | [**Map&lt;String, IoArgoprojWorkflowV1alpha1LifecycleHook&gt;**](IoArgoprojWorkflowV1alpha1LifecycleHook.md) | Hooks hold the lifecycle hook which is invoked at lifecycle of task, irrespective of the success, failure, or error status of the primary task |  [optional]
**inline** | [**IoArgoprojWorkflowV1alpha1Template**](IoArgoprojWorkflowV1alpha1Template.md) |  |  [optional]
**name** | **String** | Name is the name of the target | 
**onExit** | **String** | OnExit is a template reference which is invoked at the end of the template, irrespective of the success, failure, or error of the primary template. DEPRECATED: Use Hooks[exit].Template instead. |  [optional]
**template** | **String** | Name of template to execute |  [optional]
**templateRef** | [**IoArgoprojWorkflowV1alpha1TemplateRef**](IoArgoprojWorkflowV1alpha1TemplateRef.md) |  |  [optional]
**when** | **String** | When is an expression in which the task should conditionally execute |  [optional]
**withItems** | **List&lt;Object&gt;** | WithItems expands a task into multiple parallel tasks from the items in the list Note: The structure of WithItems is free-form, so we need \&quot;x-kubernetes-preserve-unknown-fields: true\&quot; in the validation schema. |  [optional]
**withParam** | **String** | WithParam expands a task into multiple parallel tasks from the value in the parameter, which is expected to be a JSON list. |  [optional]
**withSequence** | [**IoArgoprojWorkflowV1alpha1Sequence**](IoArgoprojWorkflowV1alpha1Sequence.md) |  |  [optional]



