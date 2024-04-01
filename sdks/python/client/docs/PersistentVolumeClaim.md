# PersistentVolumeClaim

PersistentVolumeClaim is a user's request for and claim to a persistent volume

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **str** | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources | [optional] 
**kind** | **str** | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds | [optional] 
**metadata** | [**ObjectMeta**](ObjectMeta.md) |  | [optional] 
**spec** | [**PersistentVolumeClaimSpec**](PersistentVolumeClaimSpec.md) |  | [optional] 
**status** | [**PersistentVolumeClaimStatus**](PersistentVolumeClaimStatus.md) |  | [optional] 

## Example

```python
from argo_workflows.models.persistent_volume_claim import PersistentVolumeClaim

# TODO update the JSON string below
json = "{}"
# create an instance of PersistentVolumeClaim from a JSON string
persistent_volume_claim_instance = PersistentVolumeClaim.from_json(json)
# print the JSON string representation of the object
print(PersistentVolumeClaim.to_json())

# convert the object into a dict
persistent_volume_claim_dict = persistent_volume_claim_instance.to_dict()
# create an instance of PersistentVolumeClaim from a dict
persistent_volume_claim_form_dict = persistent_volume_claim.from_dict(persistent_volume_claim_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


