

# V1alpha1WorkflowStep

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**arguments** | [**V1alpha1Arguments**](V1alpha1Arguments.md) |  |  [optional]
**continueOn** | [**V1alpha1ContinueOn**](V1alpha1ContinueOn.md) |  |  [optional]
**name** | **String** |  |  [optional]
**onExit** | **String** | OnExit is a template reference which is invoked at the end of the template, irrespective of the success, failure, or error of the primary template. |  [optional]
**template** | **String** |  |  [optional]
**templateRef** | [**V1alpha1TemplateRef**](V1alpha1TemplateRef.md) |  |  [optional]
**when** | **String** |  |  [optional]
**withItems** | [**List&lt;V1alpha1Item&gt;**](V1alpha1Item.md) |  |  [optional]
**withParam** | **String** | WithParam expands a step into multiple parallel steps from the value in the parameter, which is expected to be a JSON list. |  [optional]
**withSequence** | [**V1alpha1Sequence**](V1alpha1Sequence.md) |  |  [optional]



