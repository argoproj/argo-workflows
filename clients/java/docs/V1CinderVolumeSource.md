

# V1CinderVolumeSource

Represents a cinder volume resource in Openstack. A Cinder volume must exist before mounting to a container. The volume must also be in the same region as the kubelet. Cinder volumes support ownership management and SELinux relabeling.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fsType** | **String** |  |  [optional]
**readOnly** | **Boolean** |  |  [optional]
**secretRef** | [**V1LocalObjectReference**](V1LocalObjectReference.md) |  |  [optional]
**volumeID** | **String** |  |  [optional]



