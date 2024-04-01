# IoArgoprojWorkflowV1alpha1ArtifactRepository

ArtifactRepository represents an artifact repository in which a controller will store its artifacts

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**archive_logs** | **bool** | ArchiveLogs enables log archiving | [optional] 
**artifactory** | [**IoArgoprojWorkflowV1alpha1ArtifactoryArtifactRepository**](IoArgoprojWorkflowV1alpha1ArtifactoryArtifactRepository.md) |  | [optional] 
**azure** | [**IoArgoprojWorkflowV1alpha1AzureArtifactRepository**](IoArgoprojWorkflowV1alpha1AzureArtifactRepository.md) |  | [optional] 
**gcs** | [**IoArgoprojWorkflowV1alpha1GCSArtifactRepository**](IoArgoprojWorkflowV1alpha1GCSArtifactRepository.md) |  | [optional] 
**hdfs** | [**IoArgoprojWorkflowV1alpha1HDFSArtifactRepository**](IoArgoprojWorkflowV1alpha1HDFSArtifactRepository.md) |  | [optional] 
**oss** | [**IoArgoprojWorkflowV1alpha1OSSArtifactRepository**](IoArgoprojWorkflowV1alpha1OSSArtifactRepository.md) |  | [optional] 
**s3** | [**IoArgoprojWorkflowV1alpha1S3ArtifactRepository**](IoArgoprojWorkflowV1alpha1S3ArtifactRepository.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_artifact_repository import IoArgoprojWorkflowV1alpha1ArtifactRepository

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactRepository from a JSON string
io_argoproj_workflow_v1alpha1_artifact_repository_instance = IoArgoprojWorkflowV1alpha1ArtifactRepository.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ArtifactRepository.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_artifact_repository_dict = io_argoproj_workflow_v1alpha1_artifact_repository_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactRepository from a dict
io_argoproj_workflow_v1alpha1_artifact_repository_form_dict = io_argoproj_workflow_v1alpha1_artifact_repository.from_dict(io_argoproj_workflow_v1alpha1_artifact_repository_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


