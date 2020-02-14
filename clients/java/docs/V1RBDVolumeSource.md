

# V1RBDVolumeSource

Represents a Rados Block Device mount that lasts the lifetime of a pod. RBD volumes support ownership management and SELinux relabeling.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fsType** | **String** |  |  [optional]
**image** | **String** |  |  [optional]
**keyring** | **String** |  |  [optional]
**monitors** | **List&lt;String&gt;** |  |  [optional]
**pool** | **String** |  |  [optional]
**readOnly** | **Boolean** |  |  [optional]
**secretRef** | [**V1LocalObjectReference**](V1LocalObjectReference.md) |  |  [optional]
**user** | **String** |  |  [optional]



