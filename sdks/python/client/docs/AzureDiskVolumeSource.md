# AzureDiskVolumeSource

AzureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**caching_mode** | **str** | Host Caching mode: None, Read Only, Read Write. | [optional] 
**disk_name** | **str** | The Name of the data disk in the blob storage | 
**disk_uri** | **str** | The URI the data disk in the blob storage | 
**fs_type** | **str** | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. | [optional] 
**kind** | **str** | Expected values Shared: multiple blob disks per storage account  Dedicated: single blob disk per storage account  Managed: azure managed data disk (only in managed availability set). defaults to shared | [optional] 
**read_only** | **bool** | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | [optional] 

## Example

```python
from argo_workflows.models.azure_disk_volume_source import AzureDiskVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of AzureDiskVolumeSource from a JSON string
azure_disk_volume_source_instance = AzureDiskVolumeSource.from_json(json)
# print the JSON string representation of the object
print(AzureDiskVolumeSource.to_json())

# convert the object into a dict
azure_disk_volume_source_dict = azure_disk_volume_source_instance.to_dict()
# create an instance of AzureDiskVolumeSource from a dict
azure_disk_volume_source_form_dict = azure_disk_volume_source.from_dict(azure_disk_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


