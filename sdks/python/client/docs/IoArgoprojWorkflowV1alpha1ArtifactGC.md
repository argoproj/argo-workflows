# IoArgoprojWorkflowV1alpha1ArtifactGC

ArtifactGC describes how to delete artifacts from completed Workflows - this is embedded into the WorkflowLevelArtifactGC, and also used for individual Artifacts to override that as needed

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**env** | [**[EnvVar]**](EnvVar.md) | Env is an optional field for specifying environment variables that should be assigned to the Pod doing the deletion | [optional] 
**pod_metadata** | [**IoArgoprojWorkflowV1alpha1Metadata**](IoArgoprojWorkflowV1alpha1Metadata.md) |  | [optional] 
**service_account_name** | **str** | ServiceAccountName is an optional field for specifying the Service Account that should be assigned to the Pod doing the deletion | [optional] 
**strategy** | **str** | Strategy is the strategy to use. | [optional] 
**volume_mounts** | [**[VolumeMount]**](VolumeMount.md) | VolumeMounts is an optional field for specifying volume mounts that should be assigned to the Pod doing the deletion | [optional] 
**volumes** | [**[Volume]**](Volume.md) | Volumes is an optional field for specifying volumes that should be assigned to the Pod doing the deletion | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


