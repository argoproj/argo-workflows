# PersistentVolumeClaimVolumeSource

PersistentVolumeClaimVolumeSource references the user's PVC in the same namespace. This volume finds the bound PV and mounts that volume for the pod. A PersistentVolumeClaimVolumeSource is, essentially, a wrapper around another type of volume that is owned by someone else (the system).

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**claim_name** | **str** | ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims | 
**read_only** | **bool** | Will force the ReadOnly setting in VolumeMounts. Default false. | [optional] 

## Example

```python
from argo_workflows.models.persistent_volume_claim_volume_source import PersistentVolumeClaimVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of PersistentVolumeClaimVolumeSource from a JSON string
persistent_volume_claim_volume_source_instance = PersistentVolumeClaimVolumeSource.from_json(json)
# print the JSON string representation of the object
print(PersistentVolumeClaimVolumeSource.to_json())

# convert the object into a dict
persistent_volume_claim_volume_source_dict = persistent_volume_claim_volume_source_instance.to_dict()
# create an instance of PersistentVolumeClaimVolumeSource from a dict
persistent_volume_claim_volume_source_form_dict = persistent_volume_claim_volume_source.from_dict(persistent_volume_claim_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


