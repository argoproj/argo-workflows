# IoArgoprojWorkflowV1alpha1ValueFrom

ValueFrom describes a location in which to obtain the value to a parameter

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config_map_key_ref** | [**ConfigMapKeySelector**](ConfigMapKeySelector.md) |  | [optional] 
**default** | **str** | Default specifies a value to be used if retrieving the value from the specified source fails | [optional] 
**event** | **str** | Selector (https://github.com/expr-lang/expr) that is evaluated against the event to get the value of the parameter. E.g. &#x60;payload.message&#x60; | [optional] 
**expression** | **str** | Expression, if defined, is evaluated to specify the value for the parameter | [optional] 
**jq_filter** | **str** | JQFilter expression against the resource object in resource templates | [optional] 
**json_path** | **str** | JSONPath of a resource to retrieve an output parameter value from in resource templates | [optional] 
**parameter** | **str** | Parameter reference to a step or dag task in which to retrieve an output parameter value from (e.g. &#39;{{steps.mystep.outputs.myparam}}&#39;) | [optional] 
**path** | **str** | Path in the container to retrieve an output parameter value from in container templates | [optional] 
**supplied** | **object** | SuppliedValueFrom is a placeholder for a value to be filled in directly, either through the CLI, API, etc. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_value_from import IoArgoprojWorkflowV1alpha1ValueFrom

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ValueFrom from a JSON string
io_argoproj_workflow_v1alpha1_value_from_instance = IoArgoprojWorkflowV1alpha1ValueFrom.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ValueFrom.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_value_from_dict = io_argoproj_workflow_v1alpha1_value_from_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ValueFrom from a dict
io_argoproj_workflow_v1alpha1_value_from_form_dict = io_argoproj_workflow_v1alpha1_value_from.from_dict(io_argoproj_workflow_v1alpha1_value_from_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


