# PortworxVolumeSource

PortworxVolumeSource represents a Portworx volume resource.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | **str** | FSType represents the filesystem type to mount Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. | [optional] 
**read_only** | **bool** | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | [optional] 
**volume_id** | **str** | VolumeID uniquely identifies a Portworx volume | 

## Example

```python
from argo_workflows.models.portworx_volume_source import PortworxVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of PortworxVolumeSource from a JSON string
portworx_volume_source_instance = PortworxVolumeSource.from_json(json)
# print the JSON string representation of the object
print(PortworxVolumeSource.to_json())

# convert the object into a dict
portworx_volume_source_dict = portworx_volume_source_instance.to_dict()
# create an instance of PortworxVolumeSource from a dict
portworx_volume_source_form_dict = portworx_volume_source.from_dict(portworx_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


