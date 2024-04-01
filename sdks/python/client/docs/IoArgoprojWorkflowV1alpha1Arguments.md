# IoArgoprojWorkflowV1alpha1Arguments

Arguments to a template

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifacts** | [**List[IoArgoprojWorkflowV1alpha1Artifact]**](IoArgoprojWorkflowV1alpha1Artifact.md) | Artifacts is the list of artifacts to pass to the template or workflow | [optional] 
**parameters** | [**List[IoArgoprojWorkflowV1alpha1Parameter]**](IoArgoprojWorkflowV1alpha1Parameter.md) | Parameters is the list of parameters to pass to the template or workflow | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_arguments import IoArgoprojWorkflowV1alpha1Arguments

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Arguments from a JSON string
io_argoproj_workflow_v1alpha1_arguments_instance = IoArgoprojWorkflowV1alpha1Arguments.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Arguments.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_arguments_dict = io_argoproj_workflow_v1alpha1_arguments_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Arguments from a dict
io_argoproj_workflow_v1alpha1_arguments_form_dict = io_argoproj_workflow_v1alpha1_arguments.from_dict(io_argoproj_workflow_v1alpha1_arguments_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


