

# IoArgoprojWorkflowV1alpha1ResolveParameterEnumRequest

ResolveParameterEnumRequest identifies a parameter with an enumFrom on a stored object. The server reads the enumFrom from that object rather than accepting one from the client, so callers can only run lookups that a template author has defined.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**namespace** | **String** | Namespace the workflow will run in. Namespaced resources are listed here unless enumFrom sets a namespace. |  [optional]
**parameterName** | **String** | Name of the parameter whose enumFrom to resolve. |  [optional]
**parameters** | **Map&lt;String, String&gt;** | Current values of other parameters, substituted into {{io.argoproj.workflow.v1alpha1.parameters.&lt;name&gt;}} references. |  [optional]
**resourceKind** | **String** | Kind of the object holding the parameter: WorkflowTemplate, ClusterWorkflowTemplate, CronWorkflow or Workflow. |  [optional]
**resourceName** | **String** | Name of the object holding the parameter. WorkflowTemplates, CronWorkflows and Workflows are read from namespace. |  [optional]
**templateName** | **String** | Optional. Look the parameter up in this template&#39;s inputs instead of spec.arguments. |  [optional]



