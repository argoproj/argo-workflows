# V1AzureFileVolumeSource

AzureFile represents an Azure File Service mount on the host and bind mount to the pod.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**read_only** | **bool** | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | [optional] 
**secret_name** | **str** | the name of secret that contains Azure Storage Account Name and Key | 
**share_name** | **str** | Share Name | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


