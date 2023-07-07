# IoArgoprojWorkflowV1alpha1ValueFrom

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config_map_key_ref** | Option<[**crate::models::ConfigMapKeySelector**](ConfigMapKeySelector.md)> |  | [optional]
**default** | Option<**String**> | Default specifies a value to be used if retrieving the value from the specified source fails | [optional]
**event** | Option<**String**> | Selector (https://github.com/antonmedv/expr) that is evaluated against the event to get the value of the parameter. E.g. `payload.message` | [optional]
**expression** | Option<**String**> | Expression, if defined, is evaluated to specify the value for the parameter | [optional]
**jq_filter** | Option<**String**> | JQFilter expression against the resource object in resource templates | [optional]
**json_path** | Option<**String**> | JSONPath of a resource to retrieve an output parameter value from in resource templates | [optional]
**parameter** | Option<**String**> | Parameter reference to a step or dag task in which to retrieve an output parameter value from (e.g. '{{steps.mystep.outputs.myparam}}') | [optional]
**path** | Option<**String**> | Path in the container to retrieve an output parameter value from in container templates | [optional]
**supplied** | Option<[**serde_json::Value**](.md)> | SuppliedValueFrom is a placeholder for a value to be filled in directly, either through the CLI, API, etc. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


