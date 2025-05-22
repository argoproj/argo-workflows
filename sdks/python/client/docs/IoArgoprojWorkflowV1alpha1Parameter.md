# IoArgoprojWorkflowV1alpha1Parameter

Parameter indicate a passed string parameter to a service template with an optional default value

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name is the parameter name | 
**default** | **str** | Default is the default value to use for an input parameter if a value was not supplied | [optional] 
**description** | **str** | Description is the parameter description | [optional] 
**enum** | **[str]** | Enum holds a list of string values to choose from, for the actual value of the parameter | [optional] 
**global_name** | **str** | GlobalName exports an output parameter to the global scope, making it available as &#39;{{io.argoproj.workflow.v1alpha1.outputs.parameters.XXXX}} and in workflow.status.outputs.parameters | [optional] 
**value** | **str** | Value is the literal value to use for the parameter. If specified in the context of an input parameter, any passed values take precedence over the specified value | [optional] 
**value_from** | [**IoArgoprojWorkflowV1alpha1ValueFrom**](IoArgoprojWorkflowV1alpha1ValueFrom.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


