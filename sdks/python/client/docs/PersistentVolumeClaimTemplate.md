# PersistentVolumeClaimTemplate

PersistentVolumeClaimTemplate is used to produce PersistentVolumeClaim objects as part of an EphemeralVolumeSource.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**ObjectMeta**](ObjectMeta.md) |  | [optional] 
**spec** | [**PersistentVolumeClaimSpec**](PersistentVolumeClaimSpec.md) |  | 

## Example

```python
from argo_workflows.models.persistent_volume_claim_template import PersistentVolumeClaimTemplate

# TODO update the JSON string below
json = "{}"
# create an instance of PersistentVolumeClaimTemplate from a JSON string
persistent_volume_claim_template_instance = PersistentVolumeClaimTemplate.from_json(json)
# print the JSON string representation of the object
print(PersistentVolumeClaimTemplate.to_json())

# convert the object into a dict
persistent_volume_claim_template_dict = persistent_volume_claim_template_instance.to_dict()
# create an instance of PersistentVolumeClaimTemplate from a dict
persistent_volume_claim_template_form_dict = persistent_volume_claim_template.from_dict(persistent_volume_claim_template_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


