# IoArgoprojWorkflowV1alpha1Outputs

Outputs hold parameters, artifacts, and results from a step

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifacts** | [**List[IoArgoprojWorkflowV1alpha1Artifact]**](IoArgoprojWorkflowV1alpha1Artifact.md) | Artifacts holds the list of output artifacts produced by a step | [optional] 
**exit_code** | **str** | ExitCode holds the exit code of a script template | [optional] 
**parameters** | [**List[IoArgoprojWorkflowV1alpha1Parameter]**](IoArgoprojWorkflowV1alpha1Parameter.md) | Parameters holds the list of output parameters produced by a step | [optional] 
**result** | **str** | Result holds the result (stdout) of a script template | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_outputs import IoArgoprojWorkflowV1alpha1Outputs

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Outputs from a JSON string
io_argoproj_workflow_v1alpha1_outputs_instance = IoArgoprojWorkflowV1alpha1Outputs.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Outputs.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_outputs_dict = io_argoproj_workflow_v1alpha1_outputs_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Outputs from a dict
io_argoproj_workflow_v1alpha1_outputs_form_dict = io_argoproj_workflow_v1alpha1_outputs.from_dict(io_argoproj_workflow_v1alpha1_outputs_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


