# V1ScaleIOVolumeSource

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | **str** |  | [optional] 
**gateway** | **str** | The host address of the ScaleIO API Gateway. | [optional] 
**protection_domain** | **str** |  | [optional] 
**read_only** | **bool** |  | [optional] 
**secret_ref** | [**V1LocalObjectReference**](V1LocalObjectReference.md) |  | [optional] 
**ssl_enabled** | **bool** |  | [optional] 
**storage_mode** | **str** |  | [optional] 
**storage_pool** | **str** |  | [optional] 
**system** | **str** | The name of the storage system as configured in ScaleIO. | [optional] 
**volume_name** | **str** | The name of a volume already created in the ScaleIO system that is associated with this volume source. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


