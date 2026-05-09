

# IoArgoprojWorkflowV1alpha1LifecycleHook


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**arguments** | [**IoArgoprojWorkflowV1alpha1Arguments**](IoArgoprojWorkflowV1alpha1Arguments.md) |  |  [optional]
**expression** | **String** | Expression is a condition expression that, when it evaluates to true, causes the hook to fire. The hook is invoked once per matching event and runs in parallel to the step or template it is attached to. Available variables depend on the hook scope (e.g. workflow.status, steps.status, tasks.status). |  [optional]
**template** | **String** | Template is the name of the template to execute by the hook |  [optional]
**templateRef** | [**IoArgoprojWorkflowV1alpha1TemplateRef**](IoArgoprojWorkflowV1alpha1TemplateRef.md) |  |  [optional]



