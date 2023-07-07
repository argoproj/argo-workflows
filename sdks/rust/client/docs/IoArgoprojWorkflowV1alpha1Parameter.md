# IoArgoprojWorkflowV1alpha1Parameter

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**default** | Option<**String**> | Default is the default value to use for an input parameter if a value was not supplied | [optional]
**description** | Option<**String**> | Description is the parameter description | [optional]
**_enum** | Option<**Vec<String>**> | Enum holds a list of string values to choose from, for the actual value of the parameter | [optional]
**global_name** | Option<**String**> | GlobalName exports an output parameter to the global scope, making it available as '{{io.argoproj.workflow.v1alpha1.outputs.parameters.XXXX}} and in workflow.status.outputs.parameters | [optional]
**name** | **String** | Name is the parameter name | 
**value** | Option<**String**> | Value is the literal value to use for the parameter. If specified in the context of an input parameter, the value takes precedence over any passed values | [optional]
**value_from** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ValueFrom**](io.argoproj.workflow.v1alpha1.ValueFrom.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


