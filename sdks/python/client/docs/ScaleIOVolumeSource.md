# ScaleIOVolumeSource

ScaleIOVolumeSource represents a persistent ScaleIO volume

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | **str** | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Default is \&quot;xfs\&quot;. | [optional] 
**gateway** | **str** | The host address of the ScaleIO API Gateway. | 
**protection_domain** | **str** | The name of the ScaleIO Protection Domain for the configured storage. | [optional] 
**read_only** | **bool** | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | [optional] 
**secret_ref** | [**LocalObjectReference**](LocalObjectReference.md) |  | 
**ssl_enabled** | **bool** | Flag to enable/disable SSL communication with Gateway, default false | [optional] 
**storage_mode** | **str** | Indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned. Default is ThinProvisioned. | [optional] 
**storage_pool** | **str** | The ScaleIO Storage Pool associated with the protection domain. | [optional] 
**system** | **str** | The name of the storage system as configured in ScaleIO. | 
**volume_name** | **str** | The name of a volume already created in the ScaleIO system that is associated with this volume source. | [optional] 

## Example

```python
from argo_workflows.models.scale_io_volume_source import ScaleIOVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of ScaleIOVolumeSource from a JSON string
scale_io_volume_source_instance = ScaleIOVolumeSource.from_json(json)
# print the JSON string representation of the object
print(ScaleIOVolumeSource.to_json())

# convert the object into a dict
scale_io_volume_source_dict = scale_io_volume_source_instance.to_dict()
# create an instance of ScaleIOVolumeSource from a dict
scale_io_volume_source_form_dict = scale_io_volume_source.from_dict(scale_io_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


