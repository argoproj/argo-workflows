

# VsphereVirtualDiskVolumeSource

Represents a vSphere volume resource.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fsType** | **String** | fsType is filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. |  [optional]
**storagePolicyID** | **String** | storagePolicyID is the storage Policy Based Management (SPBM) profile ID associated with the StoragePolicyName. |  [optional]
**storagePolicyName** | **String** | storagePolicyName is the storage Policy Based Management (SPBM) profile name. |  [optional]
**volumePath** | **String** | volumePath is the path that identifies vSphere volume vmdk | 



