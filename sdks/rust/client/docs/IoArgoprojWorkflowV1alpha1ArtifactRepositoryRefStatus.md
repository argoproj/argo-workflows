# IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifact_repository** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ArtifactRepository**](io.argoproj.workflow.v1alpha1.ArtifactRepository.md)> |  | [optional]
**config_map** | Option<**String**> | The name of the config map. Defaults to \"artifact-repositories\". | [optional]
**default** | Option<**bool**> | If this ref represents the default artifact repository, rather than a config map. | [optional]
**key** | Option<**String**> | The config map key. Defaults to the value of the \"workflows.argoproj.io/default-artifact-repository\" annotation. | [optional]
**namespace** | Option<**String**> | The namespace of the config map. Defaults to the workflow's namespace, or the controller's namespace (if found). | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


