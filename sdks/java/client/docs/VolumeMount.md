

# VolumeMount

VolumeMount describes a mounting of a Volume within a container.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**mountPath** | **String** | Path within the container at which the volume should be mounted.  Must not contain &#39;:&#39;. | 
**mountPropagation** | **String** | mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10. |  [optional]
**name** | **String** | This must match the Name of a Volume. | 
**readOnly** | **Boolean** | Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false. |  [optional]
**subPath** | **String** | Path within the volume from which the container&#39;s volume should be mounted. Defaults to \&quot;\&quot; (volume&#39;s root). |  [optional]
**subPathExpr** | **String** | Expanded path within the volume from which the container&#39;s volume should be mounted. Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container&#39;s environment. Defaults to \&quot;\&quot; (volume&#39;s root). SubPathExpr and SubPath are mutually exclusive. |  [optional]



