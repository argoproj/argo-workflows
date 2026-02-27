

# IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef

ArtifactRepositoryRef is a reference to an artifact repository config map.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**configMap** | **String** | The name of the config map. Defaults to \&quot;artifact-repositories\&quot;. |  [optional]
**key** | **String** | The config map key. Defaults to the value of the \&quot;workflows.argoproj.io/default-artifact-repository\&quot; annotation. |  [optional]



