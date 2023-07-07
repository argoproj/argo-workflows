# ScaleIoVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | Option<**String**> | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". Default is \"xfs\". | [optional]
**gateway** | **String** | The host address of the ScaleIO API Gateway. | 
**protection_domain** | Option<**String**> | The name of the ScaleIO Protection Domain for the configured storage. | [optional]
**read_only** | Option<**bool**> | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | [optional]
**secret_ref** | [**crate::models::LocalObjectReference**](LocalObjectReference.md) |  | 
**ssl_enabled** | Option<**bool**> | Flag to enable/disable SSL communication with Gateway, default false | [optional]
**storage_mode** | Option<**String**> | Indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned. Default is ThinProvisioned. | [optional]
**storage_pool** | Option<**String**> | The ScaleIO Storage Pool associated with the protection domain. | [optional]
**system** | **String** | The name of the storage system as configured in ScaleIO. | 
**volume_name** | Option<**String**> | The name of a volume already created in the ScaleIO system that is associated with this volume source. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


