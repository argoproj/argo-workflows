# VolumeProjection

Projection that may be projected along with other supported volume types

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config_map** | [**ConfigMapProjection**](ConfigMapProjection.md) |  | [optional] 
**downward_api** | [**DownwardAPIProjection**](DownwardAPIProjection.md) |  | [optional] 
**secret** | [**SecretProjection**](SecretProjection.md) |  | [optional] 
**service_account_token** | [**ServiceAccountTokenProjection**](ServiceAccountTokenProjection.md) |  | [optional] 

## Example

```python
from argo_workflows.models.volume_projection import VolumeProjection

# TODO update the JSON string below
json = "{}"
# create an instance of VolumeProjection from a JSON string
volume_projection_instance = VolumeProjection.from_json(json)
# print the JSON string representation of the object
print(VolumeProjection.to_json())

# convert the object into a dict
volume_projection_dict = volume_projection_instance.to_dict()
# create an instance of VolumeProjection from a dict
volume_projection_form_dict = volume_projection.from_dict(volume_projection_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


