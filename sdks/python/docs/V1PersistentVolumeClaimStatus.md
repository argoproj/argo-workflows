# V1PersistentVolumeClaimStatus

PersistentVolumeClaimStatus is the current status of a persistent volume claim.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_modes** | **list[str]** | AccessModes contains the actual access modes the volume backing the PVC has. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1 | [optional] 
**capacity** | **dict(str, str)** | Represents the actual resources of the underlying volume. | [optional] 
**conditions** | [**list[V1PersistentVolumeClaimCondition]**](V1PersistentVolumeClaimCondition.md) | Current Condition of persistent volume claim. If underlying persistent volume is being resized then the Condition will be set to &#39;ResizeStarted&#39;. | [optional] 
**phase** | **str** | Phase represents the current phase of PersistentVolumeClaim. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


