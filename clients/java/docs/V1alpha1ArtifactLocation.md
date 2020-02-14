

# V1alpha1ArtifactLocation

ArtifactLocation describes a location for a single or multiple artifacts. It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname). It is also used to describe the location of multiple artifacts such as the archive location of a single workflow step, which the executor will use as a default location to store its files.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**archiveLogs** | **Boolean** |  |  [optional]
**artifactory** | [**V1alpha1ArtifactoryArtifact**](V1alpha1ArtifactoryArtifact.md) |  |  [optional]
**git** | [**V1alpha1GitArtifact**](V1alpha1GitArtifact.md) |  |  [optional]
**hdfs** | [**V1alpha1HDFSArtifact**](V1alpha1HDFSArtifact.md) |  |  [optional]
**http** | [**V1alpha1HTTPArtifact**](V1alpha1HTTPArtifact.md) |  |  [optional]
**raw** | [**V1alpha1RawArtifact**](V1alpha1RawArtifact.md) |  |  [optional]
**s3** | [**V1alpha1S3Artifact**](V1alpha1S3Artifact.md) |  |  [optional]



