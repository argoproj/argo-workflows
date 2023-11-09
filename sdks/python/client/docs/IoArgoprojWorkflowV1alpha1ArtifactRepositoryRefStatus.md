# IoArgoprojWorkflowV1alpha1ArtifactRepositoryRefStatus


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifact_repository** | [**IoArgoprojWorkflowV1alpha1ArtifactRepository**](IoArgoprojWorkflowV1alpha1ArtifactRepository.md) |  | [optional] 
**config_map** | **str** | The name of the config map. Defaults to \&quot;artifact-repositories\&quot;. | [optional] 
**default** | **bool** | If this ref represents the default artifact repository, rather than a config map. | [optional] 
**key** | **str** | The config map key. Defaults to the value of the \&quot;workflows.argoproj.io/default-artifact-repository\&quot; annotation. | [optional] 
**namespace** | **str** | The namespace of the config map. Defaults to the workflow&#39;s namespace, or the controller&#39;s namespace (if found). | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


