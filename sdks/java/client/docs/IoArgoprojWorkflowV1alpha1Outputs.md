

# IoArgoprojWorkflowV1alpha1Outputs

Outputs hold parameters, artifacts, and results from a step

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifactLocation** | [**IoArgoprojWorkflowV1alpha1ArtifactLocation**](IoArgoprojWorkflowV1alpha1ArtifactLocation.md) |  |  [optional]
**artifactRepositoryRef** | [**IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef**](IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef.md) |  |  [optional]
**artifacts** | [**List&lt;IoArgoprojWorkflowV1alpha1Artifact&gt;**](IoArgoprojWorkflowV1alpha1Artifact.md) | Artifacts holds the list of output artifacts produced by a step |  [optional]
**exitCode** | **String** | ExitCode holds the exit code of a script template |  [optional]
**parameters** | [**List&lt;IoArgoprojWorkflowV1alpha1Parameter&gt;**](IoArgoprojWorkflowV1alpha1Parameter.md) | Parameters holds the list of output parameters produced by a step |  [optional]
**result** | **String** | Result holds the result (stdout) of a script or container template, or the response body of an HTTP template |  [optional]



