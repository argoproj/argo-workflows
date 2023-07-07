# IoArgoprojWorkflowV1alpha1WorkflowStep

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**arguments** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Arguments**](io.argoproj.workflow.v1alpha1.Arguments.md)> |  | [optional]
**continue_on** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ContinueOn**](io.argoproj.workflow.v1alpha1.ContinueOn.md)> |  | [optional]
**hooks** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojWorkflowV1alpha1LifecycleHook>**](io.argoproj.workflow.v1alpha1.LifecycleHook.md)> | Hooks holds the lifecycle hook which is invoked at lifecycle of step, irrespective of the success, failure, or error status of the primary step | [optional]
**inline** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Template**](io.argoproj.workflow.v1alpha1.Template.md)> |  | [optional]
**name** | Option<**String**> | Name of the step | [optional]
**on_exit** | Option<**String**> | OnExit is a template reference which is invoked at the end of the template, irrespective of the success, failure, or error of the primary template. DEPRECATED: Use Hooks[exit].Template instead. | [optional]
**template** | Option<**String**> | Template is the name of the template to execute as the step | [optional]
**template_ref** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1TemplateRef**](io.argoproj.workflow.v1alpha1.TemplateRef.md)> |  | [optional]
**when** | Option<**String**> | When is an expression in which the step should conditionally execute | [optional]
**with_items** | Option<[**Vec<serde_json::Value>**](serde_json::Value.md)> | WithItems expands a step into multiple parallel steps from the items in the list | [optional]
**with_param** | Option<**String**> | WithParam expands a step into multiple parallel steps from the value in the parameter, which is expected to be a JSON list. | [optional]
**with_sequence** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Sequence**](io.argoproj.workflow.v1alpha1.Sequence.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


