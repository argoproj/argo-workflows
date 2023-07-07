# PortworxVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | Option<**String**> | FSType represents the filesystem type to mount Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\". Implicitly inferred to be \"ext4\" if unspecified. | [optional]
**read_only** | Option<**bool**> | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | [optional]
**volume_id** | **String** | VolumeID uniquely identifies a Portworx volume | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


