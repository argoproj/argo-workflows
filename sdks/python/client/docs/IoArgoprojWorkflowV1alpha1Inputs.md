# IoArgoprojWorkflowV1alpha1Inputs

Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifacts** | [**List[IoArgoprojWorkflowV1alpha1Artifact]**](IoArgoprojWorkflowV1alpha1Artifact.md) | Artifact are a list of artifacts passed as inputs | [optional] 
**parameters** | [**List[IoArgoprojWorkflowV1alpha1Parameter]**](IoArgoprojWorkflowV1alpha1Parameter.md) | Parameters are a list of parameters passed as inputs | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_inputs import IoArgoprojWorkflowV1alpha1Inputs

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Inputs from a JSON string
io_argoproj_workflow_v1alpha1_inputs_instance = IoArgoprojWorkflowV1alpha1Inputs.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Inputs.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_inputs_dict = io_argoproj_workflow_v1alpha1_inputs_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Inputs from a dict
io_argoproj_workflow_v1alpha1_inputs_form_dict = io_argoproj_workflow_v1alpha1_inputs.from_dict(io_argoproj_workflow_v1alpha1_inputs_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


