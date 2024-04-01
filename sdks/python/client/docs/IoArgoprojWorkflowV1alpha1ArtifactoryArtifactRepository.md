# IoArgoprojWorkflowV1alpha1ArtifactoryArtifactRepository

ArtifactoryArtifactRepository defines the controller configuration for an artifactory artifact repository

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key_format** | **str** | KeyFormat defines the format of how to store keys and can reference workflow variables. | [optional] 
**password_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**repo_url** | **str** | RepoURL is the url for artifactory repo. | [optional] 
**username_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_artifactory_artifact_repository import IoArgoprojWorkflowV1alpha1ArtifactoryArtifactRepository

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactoryArtifactRepository from a JSON string
io_argoproj_workflow_v1alpha1_artifactory_artifact_repository_instance = IoArgoprojWorkflowV1alpha1ArtifactoryArtifactRepository.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ArtifactoryArtifactRepository.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_artifactory_artifact_repository_dict = io_argoproj_workflow_v1alpha1_artifactory_artifact_repository_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactoryArtifactRepository from a dict
io_argoproj_workflow_v1alpha1_artifactory_artifact_repository_form_dict = io_argoproj_workflow_v1alpha1_artifactory_artifact_repository.from_dict(io_argoproj_workflow_v1alpha1_artifactory_artifact_repository_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


