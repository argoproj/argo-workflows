# VsphereVirtualDiskVolumeSource

Represents a vSphere volume resource.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | **str** | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. | [optional] 
**storage_policy_id** | **str** | Storage Policy Based Management (SPBM) profile ID associated with the StoragePolicyName. | [optional] 
**storage_policy_name** | **str** | Storage Policy Based Management (SPBM) profile name. | [optional] 
**volume_path** | **str** | Path that identifies vSphere volume vmdk | 

## Example

```python
from argo_workflows.models.vsphere_virtual_disk_volume_source import VsphereVirtualDiskVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of VsphereVirtualDiskVolumeSource from a JSON string
vsphere_virtual_disk_volume_source_instance = VsphereVirtualDiskVolumeSource.from_json(json)
# print the JSON string representation of the object
print(VsphereVirtualDiskVolumeSource.to_json())

# convert the object into a dict
vsphere_virtual_disk_volume_source_dict = vsphere_virtual_disk_volume_source_instance.to_dict()
# create an instance of VsphereVirtualDiskVolumeSource from a dict
vsphere_virtual_disk_volume_source_form_dict = vsphere_virtual_disk_volume_source.from_dict(vsphere_virtual_disk_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


