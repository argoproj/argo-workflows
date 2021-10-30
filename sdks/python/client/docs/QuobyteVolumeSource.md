# QuobyteVolumeSource

Represents a Quobyte mount that lasts the lifetime of a pod. Quobyte volumes do not support ownership management or SELinux relabeling.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**registry** | **str** | Registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes | 
**volume** | **str** | Volume is a string that references an already created Quobyte volume by name. | 
**group** | **str** | Group to map volume access to Default is no group | [optional] 
**read_only** | **bool** | ReadOnly here will force the Quobyte volume to be mounted with read-only permissions. Defaults to false. | [optional] 
**tenant** | **str** | Tenant owning the given Quobyte volume in the Backend Used with dynamically provisioned Quobyte volumes, value is set by the plugin | [optional] 
**user** | **str** | User to map volume access to Defaults to serivceaccount user | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


