# VolumeMount

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**mount_path** | **String** | Path within the container at which the volume should be mounted.  Must not contain ':'. | 
**mount_propagation** | Option<**String**> | mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10. | [optional]
**name** | **String** | This must match the Name of a Volume. | 
**read_only** | Option<**bool**> | Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false. | [optional]
**sub_path** | Option<**String**> | Path within the volume from which the container's volume should be mounted. Defaults to \"\" (volume's root). | [optional]
**sub_path_expr** | Option<**String**> | Expanded path within the volume from which the container's volume should be mounted. Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment. Defaults to \"\" (volume's root). SubPathExpr and SubPath are mutually exclusive. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


