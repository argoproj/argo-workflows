

# IoArgoprojWorkflowV1alpha1Parameter

Parameter indicate a passed string parameter to a service template with an optional default value

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**_default** | **String** | Default is the default value to use for an input parameter if a value was not supplied |  [optional]
**description** | **String** | Description is the parameter description |  [optional]
**_enum** | **List&lt;String&gt;** | Enum holds a list of string values to choose from, for the actual value of the parameter |  [optional]
**globalName** | **String** | GlobalName exports an output parameter to the global scope, making it available as &#39;{{io.argoproj.workflow.v1alpha1.outputs.parameters.XXXX}} and in workflow.status.outputs.parameters |  [optional]
**name** | **String** | Name is the parameter name | 
**value** | **String** | Value is the literal value to use for the parameter. If specified in the context of an input parameter, the value takes precedence over any passed values |  [optional]
**valueFrom** | [**IoArgoprojWorkflowV1alpha1ValueFrom**](IoArgoprojWorkflowV1alpha1ValueFrom.md) |  |  [optional]



