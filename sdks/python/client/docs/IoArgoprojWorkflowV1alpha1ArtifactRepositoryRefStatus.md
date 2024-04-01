# IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifact_repository** | [**IoArgoprojWorkflowV1alpha1ArtifactRepository**](IoArgoprojWorkflowV1alpha1ArtifactRepository.md) |  | [optional] 
**config_map** | **str** | The name of the config map. Defaults to \&quot;artifact-repositories\&quot;. | [optional] 
**default** | **bool** | If this ref represents the default artifact repository, rather than a config map. | [optional] 
**key** | **str** | The config map key. Defaults to the value of the \&quot;workflows.argoproj.io/default-artifact-repository\&quot; annotation. | [optional] 
**namespace** | **str** | The namespace of the config map. Defaults to the workflow&#39;s namespace, or the controller&#39;s namespace (if found). | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_artifact_repository_ref_status import IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus from a JSON string
io_argoproj_workflow_v1alpha1_artifact_repository_ref_status_instance = IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_artifact_repository_ref_status_dict = io_argoproj_workflow_v1alpha1_artifact_repository_ref_status_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus from a dict
io_argoproj_workflow_v1alpha1_artifact_repository_ref_status_form_dict = io_argoproj_workflow_v1alpha1_artifact_repository_ref_status.from_dict(io_argoproj_workflow_v1alpha1_artifact_repository_ref_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


