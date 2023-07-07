# QuobyteVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**group** | Option<**String**> | Group to map volume access to Default is no group | [optional]
**read_only** | Option<**bool**> | ReadOnly here will force the Quobyte volume to be mounted with read-only permissions. Defaults to false. | [optional]
**registry** | **String** | Registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes | 
**tenant** | Option<**String**> | Tenant owning the given Quobyte volume in the Backend Used with dynamically provisioned Quobyte volumes, value is set by the plugin | [optional]
**user** | Option<**String**> | User to map volume access to Defaults to serivceaccount user | [optional]
**volume** | **String** | Volume is a string that references an already created Quobyte volume by name. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


