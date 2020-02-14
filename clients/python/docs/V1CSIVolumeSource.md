# V1CSIVolumeSource

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**driver** | **str** | Driver is the name of the CSI driver that handles this volume. Consult with your admin for the correct name as registered in the cluster. | [optional] 
**fs_type** | **str** |  | [optional] 
**node_publish_secret_ref** | [**V1LocalObjectReference**](V1LocalObjectReference.md) |  | [optional] 
**read_only** | **bool** |  | [optional] 
**volume_attributes** | **dict(str, str)** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


