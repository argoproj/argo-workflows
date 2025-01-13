

# StorageOSVolumeSource

Represents a StorageOS persistent volume resource.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fsType** | **String** | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. |  [optional]
**readOnly** | **Boolean** | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. |  [optional]
**secretRef** | [**io.kubernetes.client.openapi.models.V1LocalObjectReference**](io.kubernetes.client.openapi.models.V1LocalObjectReference.md) |  |  [optional]
**volumeName** | **String** | volumeName is the human-readable name of the StorageOS volume.  Volume names are only unique within a namespace. |  [optional]
**volumeNamespace** | **String** | volumeNamespace specifies the scope of the volume within StorageOS.  If no namespace is specified then the Pod&#39;s namespace will be used.  This allows the Kubernetes name scoping to be mirrored within StorageOS for tighter integration. Set VolumeName to any name to override the default behaviour. Set to \&quot;default\&quot; if you are not using namespaces within StorageOS. Namespaces that do not pre-exist within StorageOS will be created. |  [optional]



