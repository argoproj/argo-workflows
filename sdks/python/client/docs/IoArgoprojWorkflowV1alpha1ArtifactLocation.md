# IoArgoprojWorkflowV1alpha1ArtifactLocation

ArtifactLocation describes a location for a single or multiple artifacts. It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname). It is also used to describe the location of multiple artifacts such as the archive location of a single workflow step, which the executor will use as a default location to store its files.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**archive_logs** | **bool** | ArchiveLogs indicates if the container logs should be archived | [optional] 
**artifactory** | [**IoArgoprojWorkflowV1alpha1ArtifactoryArtifact**](IoArgoprojWorkflowV1alpha1ArtifactoryArtifact.md) |  | [optional] 
**azure** | [**IoArgoprojWorkflowV1alpha1AzureArtifact**](IoArgoprojWorkflowV1alpha1AzureArtifact.md) |  | [optional] 
**gcs** | [**IoArgoprojWorkflowV1alpha1GCSArtifact**](IoArgoprojWorkflowV1alpha1GCSArtifact.md) |  | [optional] 
**git** | [**IoArgoprojWorkflowV1alpha1GitArtifact**](IoArgoprojWorkflowV1alpha1GitArtifact.md) |  | [optional] 
**hdfs** | [**IoArgoprojWorkflowV1alpha1HDFSArtifact**](IoArgoprojWorkflowV1alpha1HDFSArtifact.md) |  | [optional] 
**http** | [**IoArgoprojWorkflowV1alpha1HTTPArtifact**](IoArgoprojWorkflowV1alpha1HTTPArtifact.md) |  | [optional] 
**oss** | [**IoArgoprojWorkflowV1alpha1OSSArtifact**](IoArgoprojWorkflowV1alpha1OSSArtifact.md) |  | [optional] 
**raw** | [**IoArgoprojWorkflowV1alpha1RawArtifact**](IoArgoprojWorkflowV1alpha1RawArtifact.md) |  | [optional] 
**s3** | [**IoArgoprojWorkflowV1alpha1S3Artifact**](IoArgoprojWorkflowV1alpha1S3Artifact.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_artifact_location import IoArgoprojWorkflowV1alpha1ArtifactLocation

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactLocation from a JSON string
io_argoproj_workflow_v1alpha1_artifact_location_instance = IoArgoprojWorkflowV1alpha1ArtifactLocation.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ArtifactLocation.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_artifact_location_dict = io_argoproj_workflow_v1alpha1_artifact_location_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactLocation from a dict
io_argoproj_workflow_v1alpha1_artifact_location_form_dict = io_argoproj_workflow_v1alpha1_artifact_location.from_dict(io_argoproj_workflow_v1alpha1_artifact_location_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


