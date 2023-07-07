# CsiVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**driver** | **String** | Driver is the name of the CSI driver that handles this volume. Consult with your admin for the correct name as registered in the cluster. | 
**fs_type** | Option<**String**> | Filesystem type to mount. Ex. \"ext4\", \"xfs\", \"ntfs\". If not provided, the empty value is passed to the associated CSI driver which will determine the default filesystem to apply. | [optional]
**node_publish_secret_ref** | Option<[**crate::models::LocalObjectReference**](LocalObjectReference.md)> |  | [optional]
**read_only** | Option<**bool**> | Specifies a read-only configuration for the volume. Defaults to false (read/write). | [optional]
**volume_attributes** | Option<**::std::collections::HashMap<String, String>**> | VolumeAttributes stores driver-specific properties that are passed to the CSI driver. Consult your driver's documentation for supported values. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


