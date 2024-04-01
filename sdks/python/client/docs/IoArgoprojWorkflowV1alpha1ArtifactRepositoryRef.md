# IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config_map** | **str** | The name of the config map. Defaults to \&quot;artifact-repositories\&quot;. | [optional] 
**key** | **str** | The config map key. Defaults to the value of the \&quot;workflows.argoproj.io/default-artifact-repository\&quot; annotation. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_artifact_repository_ref import IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef from a JSON string
io_argoproj_workflow_v1alpha1_artifact_repository_ref_instance = IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_artifact_repository_ref_dict = io_argoproj_workflow_v1alpha1_artifact_repository_ref_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef from a dict
io_argoproj_workflow_v1alpha1_artifact_repository_ref_form_dict = io_argoproj_workflow_v1alpha1_artifact_repository_ref.from_dict(io_argoproj_workflow_v1alpha1_artifact_repository_ref_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


