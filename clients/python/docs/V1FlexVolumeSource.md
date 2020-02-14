# V1FlexVolumeSource

FlexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**driver** | **str** | Driver is the name of the driver to use for this volume. | [optional] 
**fs_type** | **str** |  | [optional] 
**options** | **dict(str, str)** |  | [optional] 
**read_only** | **bool** |  | [optional] 
**secret_ref** | [**V1LocalObjectReference**](V1LocalObjectReference.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


