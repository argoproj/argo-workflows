# IoArgoprojWorkflowV1alpha1Parameter

Parameter indicate a passed string parameter to a service template with an optional default value

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**default** | **str** | Default is the default value to use for an input parameter if a value was not supplied | [optional] 
**description** | **str** | Description is the parameter description | [optional] 
**enum** | **List[str]** | Enum holds a list of string values to choose from, for the actual value of the parameter | [optional] 
**global_name** | **str** | GlobalName exports an output parameter to the global scope, making it available as &#39;{{io.argoproj.workflow.v1alpha1.outputs.parameters.XXXX}} and in workflow.status.outputs.parameters | [optional] 
**name** | **str** | Name is the parameter name | 
**value** | **str** | Value is the literal value to use for the parameter. If specified in the context of an input parameter, the value takes precedence over any passed values | [optional] 
**value_from** | [**IoArgoprojWorkflowV1alpha1ValueFrom**](IoArgoprojWorkflowV1alpha1ValueFrom.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_parameter import IoArgoprojWorkflowV1alpha1Parameter

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Parameter from a JSON string
io_argoproj_workflow_v1alpha1_parameter_instance = IoArgoprojWorkflowV1alpha1Parameter.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Parameter.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_parameter_dict = io_argoproj_workflow_v1alpha1_parameter_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Parameter from a dict
io_argoproj_workflow_v1alpha1_parameter_form_dict = io_argoproj_workflow_v1alpha1_parameter.from_dict(io_argoproj_workflow_v1alpha1_parameter_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


