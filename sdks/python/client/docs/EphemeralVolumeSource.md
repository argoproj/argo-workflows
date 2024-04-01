# EphemeralVolumeSource

Represents an ephemeral volume that is handled by a normal storage driver.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**volume_claim_template** | [**PersistentVolumeClaimTemplate**](PersistentVolumeClaimTemplate.md) |  | [optional] 

## Example

```python
from argo_workflows.models.ephemeral_volume_source import EphemeralVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of EphemeralVolumeSource from a JSON string
ephemeral_volume_source_instance = EphemeralVolumeSource.from_json(json)
# print the JSON string representation of the object
print(EphemeralVolumeSource.to_json())

# convert the object into a dict
ephemeral_volume_source_dict = ephemeral_volume_source_instance.to_dict()
# create an instance of EphemeralVolumeSource from a dict
ephemeral_volume_source_form_dict = ephemeral_volume_source.from_dict(ephemeral_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


