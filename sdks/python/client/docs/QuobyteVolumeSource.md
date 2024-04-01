# QuobyteVolumeSource

Represents a Quobyte mount that lasts the lifetime of a pod. Quobyte volumes do not support ownership management or SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**group** | **str** | Group to map volume access to Default is no group | [optional] 
**read_only** | **bool** | ReadOnly here will force the Quobyte volume to be mounted with read-only permissions. Defaults to false. | [optional] 
**registry** | **str** | Registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes | 
**tenant** | **str** | Tenant owning the given Quobyte volume in the Backend Used with dynamically provisioned Quobyte volumes, value is set by the plugin | [optional] 
**user** | **str** | User to map volume access to Defaults to serivceaccount user | [optional] 
**volume** | **str** | Volume is a string that references an already created Quobyte volume by name. | 

## Example

```python
from argo_workflows.models.quobyte_volume_source import QuobyteVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of QuobyteVolumeSource from a JSON string
quobyte_volume_source_instance = QuobyteVolumeSource.from_json(json)
# print the JSON string representation of the object
print(QuobyteVolumeSource.to_json())

# convert the object into a dict
quobyte_volume_source_dict = quobyte_volume_source_instance.to_dict()
# create an instance of QuobyteVolumeSource from a dict
quobyte_volume_source_form_dict = quobyte_volume_source.from_dict(quobyte_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


