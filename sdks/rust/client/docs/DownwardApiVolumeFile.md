# DownwardApiVolumeFile

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**field_ref** | Option<[**crate::models::ObjectFieldSelector**](ObjectFieldSelector.md)> |  | [optional]
**mode** | Option<**i32**> | Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. | [optional]
**path** | **String** | Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..' | 
**resource_field_ref** | Option<[**crate::models::ResourceFieldSelector**](ResourceFieldSelector.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


