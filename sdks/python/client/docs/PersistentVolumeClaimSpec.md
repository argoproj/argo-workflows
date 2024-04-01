# PersistentVolumeClaimSpec

PersistentVolumeClaimSpec describes the common attributes of storage devices and allows a Source for provider-specific attributes

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_modes** | **List[str]** | AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1 | [optional] 
**data_source** | [**TypedLocalObjectReference**](TypedLocalObjectReference.md) |  | [optional] 
**data_source_ref** | [**TypedLocalObjectReference**](TypedLocalObjectReference.md) |  | [optional] 
**resources** | [**ResourceRequirements**](ResourceRequirements.md) |  | [optional] 
**selector** | [**LabelSelector**](LabelSelector.md) |  | [optional] 
**storage_class_name** | **str** | Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1 | [optional] 
**volume_mode** | **str** | volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec. | [optional] 
**volume_name** | **str** | VolumeName is the binding reference to the PersistentVolume backing this claim. | [optional] 

## Example

```python
from argo_workflows.models.persistent_volume_claim_spec import PersistentVolumeClaimSpec

# TODO update the JSON string below
json = "{}"
# create an instance of PersistentVolumeClaimSpec from a JSON string
persistent_volume_claim_spec_instance = PersistentVolumeClaimSpec.from_json(json)
# print the JSON string representation of the object
print(PersistentVolumeClaimSpec.to_json())

# convert the object into a dict
persistent_volume_claim_spec_dict = persistent_volume_claim_spec_instance.to_dict()
# create an instance of PersistentVolumeClaimSpec from a dict
persistent_volume_claim_spec_form_dict = persistent_volume_claim_spec.from_dict(persistent_volume_claim_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


