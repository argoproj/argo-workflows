# V1PersistentVolumeClaimStatus

PersistentVolumeClaimStatus is the current status of a persistent volume claim.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_modes** | **list[str]** |  | [optional] 
**capacity** | [**dict(str, ResourceQuantity)**](ResourceQuantity.md) |  | [optional] 
**conditions** | [**list[V1PersistentVolumeClaimCondition]**](V1PersistentVolumeClaimCondition.md) |  | [optional] 
**phase** | **str** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


