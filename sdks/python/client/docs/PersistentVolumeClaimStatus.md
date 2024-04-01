# PersistentVolumeClaimStatus

PersistentVolumeClaimStatus is the current status of a persistent volume claim.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_modes** | **List[str]** | AccessModes contains the actual access modes the volume backing the PVC has. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1 | [optional] 
**allocated_resources** | **Dict[str, str]** | The storage resource within AllocatedResources tracks the capacity allocated to a PVC. It may be larger than the actual capacity when a volume expansion operation is requested. For storage quota, the larger value from allocatedResources and PVC.spec.resources is used. If allocatedResources is not set, PVC.spec.resources alone is used for quota calculation. If a volume expansion capacity request is lowered, allocatedResources is only lowered if there are no expansion operations in progress and if the actual volume capacity is equal or lower than the requested capacity. This is an alpha field and requires enabling RecoverVolumeExpansionFailure feature. | [optional] 
**capacity** | **Dict[str, str]** | Represents the actual resources of the underlying volume. | [optional] 
**conditions** | [**List[PersistentVolumeClaimCondition]**](PersistentVolumeClaimCondition.md) | Current Condition of persistent volume claim. If underlying persistent volume is being resized then the Condition will be set to &#39;ResizeStarted&#39;. | [optional] 
**phase** | **str** | Phase represents the current phase of PersistentVolumeClaim.  Possible enum values:  - &#x60;\&quot;Bound\&quot;&#x60; used for PersistentVolumeClaims that are bound  - &#x60;\&quot;Lost\&quot;&#x60; used for PersistentVolumeClaims that lost their underlying PersistentVolume. The claim was bound to a PersistentVolume and this volume does not exist any longer and all data on it was lost.  - &#x60;\&quot;Pending\&quot;&#x60; used for PersistentVolumeClaims that are not yet bound | [optional] 
**resize_status** | **str** | ResizeStatus stores status of resize operation. ResizeStatus is not set by default but when expansion is complete resizeStatus is set to empty string by resize controller or kubelet. This is an alpha field and requires enabling RecoverVolumeExpansionFailure feature. | [optional] 

## Example

```python
from argo_workflows.models.persistent_volume_claim_status import PersistentVolumeClaimStatus

# TODO update the JSON string below
json = "{}"
# create an instance of PersistentVolumeClaimStatus from a JSON string
persistent_volume_claim_status_instance = PersistentVolumeClaimStatus.from_json(json)
# print the JSON string representation of the object
print(PersistentVolumeClaimStatus.to_json())

# convert the object into a dict
persistent_volume_claim_status_dict = persistent_volume_claim_status_instance.to_dict()
# create an instance of PersistentVolumeClaimStatus from a dict
persistent_volume_claim_status_form_dict = persistent_volume_claim_status.from_dict(persistent_volume_claim_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


