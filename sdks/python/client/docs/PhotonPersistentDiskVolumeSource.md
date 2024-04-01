# PhotonPersistentDiskVolumeSource

Represents a Photon Controller persistent disk resource.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | **str** | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. | [optional] 
**pd_id** | **str** | ID that identifies Photon Controller persistent disk | 

## Example

```python
from argo_workflows.models.photon_persistent_disk_volume_source import PhotonPersistentDiskVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of PhotonPersistentDiskVolumeSource from a JSON string
photon_persistent_disk_volume_source_instance = PhotonPersistentDiskVolumeSource.from_json(json)
# print the JSON string representation of the object
print(PhotonPersistentDiskVolumeSource.to_json())

# convert the object into a dict
photon_persistent_disk_volume_source_dict = photon_persistent_disk_volume_source_instance.to_dict()
# create an instance of PhotonPersistentDiskVolumeSource from a dict
photon_persistent_disk_volume_source_form_dict = photon_persistent_disk_volume_source.from_dict(photon_persistent_disk_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


