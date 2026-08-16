

# IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus

ArtifactRepositoryRefStatus is the resolved artifact repository reference with namespace io.argoproj.workflow.v1alpha1.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifactRepository** | [**IoArgoprojWorkflowV1alpha1ArtifactRepository**](IoArgoprojWorkflowV1alpha1ArtifactRepository.md) |  |  [optional]
**configMap** | **String** | The name of the config map. Defaults to \&quot;artifact-repositories\&quot;. |  [optional]
**_default** | **Boolean** | If this ref represents the default artifact repository, rather than a config map. |  [optional]
**key** | **String** | The config map key. Defaults to the value of the \&quot;workflows.argoproj.io/default-artifact-repository\&quot; annotation. |  [optional]
**namespace** | **String** | The namespace of the config map. Defaults to the workflow&#39;s namespace, or the controller&#39;s namespace (if found). |  [optional]



