# V1alpha1WorkflowStep

WorkflowStep is a reference to a template to execute in a series of step
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**arguments** | [**V1alpha1Arguments**](V1alpha1Arguments.md) |  | [optional] 
**continue_on** | [**V1alpha1ContinueOn**](V1alpha1ContinueOn.md) |  | [optional] 
**name** | **str** | Name of the step | [optional] 
**on_exit** | **str** | OnExit is a template reference which is invoked at the end of the template, irrespective of the success, failure, or error of the primary template. | [optional] 
**template** | **str** | Template is the name of the template to execute as the step | [optional] 
**template_ref** | [**V1alpha1TemplateRef**](V1alpha1TemplateRef.md) |  | [optional] 
**when** | **str** | When is an expression in which the step should conditionally execute | [optional] 
**with_items** | **list[object]** | WithItems expands a step into multiple parallel steps from the items in the list | [optional] 
**with_param** | **str** | WithParam expands a step into multiple parallel steps from the value in the parameter, which is expected to be a JSON list. | [optional] 
**with_sequence** | [**V1alpha1Sequence**](V1alpha1Sequence.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


