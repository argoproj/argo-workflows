# AzureDiskVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**caching_mode** | Option<**String**> | Host Caching mode: None, Read Only, Read Write. | [optional]
**disk_name** | **String** | The Name of the data disk in the blob storage | 
**disk_uri** | **String** | The URI the data disk in the blob storage | 
**fs_type** | Option<**String**> | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. | [optional]
**kind** | Option<**String**> | Expected values Shared: multiple blob disks per storage account  Dedicated: single blob disk per storage account  Managed: azure managed data disk (only in managed availability set). defaults to shared | [optional]
**read_only** | Option<**bool**> | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


