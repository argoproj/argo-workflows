

# V1FCVolumeSource

Represents a Fibre Channel volume. Fibre Channel volumes can only be mounted as read/write once. Fibre Channel volumes support ownership management and SELinux relabeling.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fsType** | **String** |  |  [optional]
**lun** | **Integer** |  |  [optional]
**readOnly** | **Boolean** |  |  [optional]
**targetWWNs** | **List&lt;String&gt;** |  |  [optional]
**wwids** | **List&lt;String&gt;** |  |  [optional]



