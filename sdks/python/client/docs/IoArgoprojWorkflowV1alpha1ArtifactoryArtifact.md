# IoArgoprojWorkflowV1alpha1ArtifactoryArtifact

ArtifactoryArtifact is the location of an artifactory artifact

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**password_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**url** | **str** | URL of the artifact | 
**username_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_artifactory_artifact import IoArgoprojWorkflowV1alpha1ArtifactoryArtifact

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactoryArtifact from a JSON string
io_argoproj_workflow_v1alpha1_artifactory_artifact_instance = IoArgoprojWorkflowV1alpha1ArtifactoryArtifact.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ArtifactoryArtifact.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_artifactory_artifact_dict = io_argoproj_workflow_v1alpha1_artifactory_artifact_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactoryArtifact from a dict
io_argoproj_workflow_v1alpha1_artifactory_artifact_form_dict = io_argoproj_workflow_v1alpha1_artifactory_artifact.from_dict(io_argoproj_workflow_v1alpha1_artifactory_artifact_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


