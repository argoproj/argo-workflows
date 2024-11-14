

# QuobyteVolumeSource

Represents a Quobyte mount that lasts the lifetime of a pod. Quobyte volumes do not support ownership management or SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**group** | **String** | group to map volume access to Default is no group |  [optional]
**readOnly** | **Boolean** | readOnly here will force the Quobyte volume to be mounted with read-only permissions. Defaults to false. |  [optional]
**registry** | **String** | registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes | 
**tenant** | **String** | tenant owning the given Quobyte volume in the Backend Used with dynamically provisioned Quobyte volumes, value is set by the plugin |  [optional]
**user** | **String** | user to map volume access to Defaults to serivceaccount user |  [optional]
**volume** | **String** | volume is a string that references an already created Quobyte volume by name. | 



