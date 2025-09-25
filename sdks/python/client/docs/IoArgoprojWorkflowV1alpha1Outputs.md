# IoArgoprojWorkflowV1alpha1Outputs

Outputs hold parameters, artifacts, and results from a step

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifact_location** | [**IoArgoprojWorkflowV1alpha1ArtifactLocation**](IoArgoprojWorkflowV1alpha1ArtifactLocation.md) |  | [optional] 
**artifact_repository_ref** | [**IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef**](IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef.md) |  | [optional] 
**artifacts** | [**[IoArgoprojWorkflowV1alpha1Artifact]**](IoArgoprojWorkflowV1alpha1Artifact.md) | Artifacts holds the list of output artifacts produced by a step | [optional] 
**exit_code** | **str** | ExitCode holds the exit code of a script template | [optional] 
**parameters** | [**[IoArgoprojWorkflowV1alpha1Parameter]**](IoArgoprojWorkflowV1alpha1Parameter.md) | Parameters holds the list of output parameters produced by a step | [optional] 
**result** | **str** | Result holds the result (stdout) of a script or container template, or the response body of an HTTP template | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


