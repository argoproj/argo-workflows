# V1QuobyteVolumeSource

Represents a Quobyte mount that lasts the lifetime of a pod. Quobyte volumes do not support ownership management or SELinux relabeling.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**group** | **str** |  | [optional] 
**read_only** | **bool** |  | [optional] 
**registry** | **str** |  | [optional] 
**tenant** | **str** |  | [optional] 
**user** | **str** |  | [optional] 
**volume** | **str** | Volume is a string that references an already created Quobyte volume by name. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


