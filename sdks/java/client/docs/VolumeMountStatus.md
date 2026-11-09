

# VolumeMountStatus

VolumeMountStatus shows status of volume mounts.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**mountPath** | **String** | MountPath corresponds to the original VolumeMount. | 
**name** | **String** | Name corresponds to the name of the original VolumeMount. | 
**readOnly** | **Boolean** | ReadOnly corresponds to the original VolumeMount. |  [optional]
**recursiveReadOnly** | **String** | RecursiveReadOnly must be set to Disabled, Enabled, or unspecified (for non-readonly mounts). An IfPossible value in the original VolumeMount must be translated to Disabled or Enabled, depending on the mount result. |  [optional]



