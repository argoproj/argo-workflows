# PortworxVolumeSource

PortworxVolumeSource represents a Portworx volume resource.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**volume_id** | **str** | volumeID uniquely identifies a Portworx volume | 
**fs_type** | **str** | fSType represents the filesystem type to mount Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. | [optional] 
**read_only** | **bool** | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


