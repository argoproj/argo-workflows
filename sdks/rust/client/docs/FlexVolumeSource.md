# FlexVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**driver** | **String** | Driver is the name of the driver to use for this volume. | 
**fs_type** | Option<**String**> | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". The default filesystem depends on FlexVolume script. | [optional]
**options** | Option<**::std::collections::HashMap<String, String>**> | Optional: Extra command options if any. | [optional]
**read_only** | Option<**bool**> | Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | [optional]
**secret_ref** | Option<[**crate::models::LocalObjectReference**](LocalObjectReference.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


