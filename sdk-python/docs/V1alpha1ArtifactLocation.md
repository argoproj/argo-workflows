# V1alpha1ArtifactLocation

ArtifactLocation describes a location for a single or multiple artifacts. It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname). It is also used to describe the location of multiple artifacts such as the archive location of a single workflow step, which the executor will use as a default location to store its files.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**archive_logs** | **bool** | ArchiveLogs indicates if the container logs should be archived | [optional] 
**artifactory** | [**V1alpha1ArtifactoryArtifact**](V1alpha1ArtifactoryArtifact.md) |  | [optional] 
**gcs** | [**V1alpha1GCSArtifact**](V1alpha1GCSArtifact.md) |  | [optional] 
**git** | [**V1alpha1GitArtifact**](V1alpha1GitArtifact.md) |  | [optional] 
**hdfs** | [**V1alpha1HDFSArtifact**](V1alpha1HDFSArtifact.md) |  | [optional] 
**http** | [**V1alpha1HTTPArtifact**](V1alpha1HTTPArtifact.md) |  | [optional] 
**oss** | [**V1alpha1OSSArtifact**](V1alpha1OSSArtifact.md) |  | [optional] 
**raw** | [**V1alpha1RawArtifact**](V1alpha1RawArtifact.md) |  | [optional] 
**s3** | [**V1alpha1S3Artifact**](V1alpha1S3Artifact.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


