# IoArgoprojWorkflowV1alpha1Inputs

Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifact_location** | [**IoArgoprojWorkflowV1alpha1ArtifactLocation**](IoArgoprojWorkflowV1alpha1ArtifactLocation.md) |  | [optional] 
**artifact_repository_ref** | [**IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef**](IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef.md) |  | [optional] 
**artifacts** | [**[IoArgoprojWorkflowV1alpha1Artifact]**](IoArgoprojWorkflowV1alpha1Artifact.md) | Artifact are a list of artifacts passed as inputs | [optional] 
**parameters** | [**[IoArgoprojWorkflowV1alpha1Parameter]**](IoArgoprojWorkflowV1alpha1Parameter.md) | Parameters are a list of parameters passed as inputs | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


