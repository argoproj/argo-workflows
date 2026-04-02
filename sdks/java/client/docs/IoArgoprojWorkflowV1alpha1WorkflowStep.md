

# IoArgoprojWorkflowV1alpha1WorkflowStep

WorkflowStep is a reference to a template to execute in a series of step Note: CEL validation cannot check withItems (Schemaless) or inline (PreserveUnknownFields) fields.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**arguments** | [**IoArgoprojWorkflowV1alpha1Arguments**](IoArgoprojWorkflowV1alpha1Arguments.md) |  |  [optional]
**continueOn** | [**IoArgoprojWorkflowV1alpha1ContinueOn**](IoArgoprojWorkflowV1alpha1ContinueOn.md) |  |  [optional]
**hooks** | [**Map&lt;String, IoArgoprojWorkflowV1alpha1LifecycleHook&gt;**](IoArgoprojWorkflowV1alpha1LifecycleHook.md) | Hooks holds the lifecycle hook which is invoked at lifecycle of step, irrespective of the success, failure, or error status of the primary step |  [optional]
**inline** | [**IoArgoprojWorkflowV1alpha1Template**](IoArgoprojWorkflowV1alpha1Template.md) |  |  [optional]
**name** | **String** | Name of the step |  [optional]
**onExit** | **String** | OnExit is a template reference which is invoked at the end of the template, irrespective of the success, failure, or error of the primary template.  Deprecated: Use Hooks[exit].Template instead. |  [optional]
**template** | **String** | Template is the name of the template to execute as the step |  [optional]
**templateRef** | [**IoArgoprojWorkflowV1alpha1TemplateRef**](IoArgoprojWorkflowV1alpha1TemplateRef.md) |  |  [optional]
**when** | **String** | When is an expression in which the step should conditionally execute |  [optional]
**withItems** | **List&lt;Object&gt;** | WithItems expands a step into multiple parallel steps from the items in the list Note: The structure of WithItems is free-form, so we need \&quot;x-kubernetes-preserve-unknown-fields: true\&quot; in the validation schema. |  [optional]
**withParam** | **String** | WithParam expands a step into multiple parallel steps from the value in the parameter, which is expected to be a JSON list. |  [optional]
**withSequence** | [**IoArgoprojWorkflowV1alpha1Sequence**](IoArgoprojWorkflowV1alpha1Sequence.md) |  |  [optional]



