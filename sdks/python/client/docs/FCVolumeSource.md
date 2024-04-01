# FCVolumeSource

Represents a Fibre Channel volume. Fibre Channel volumes can only be mounted as read/write once. Fibre Channel volumes support ownership management and SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | **str** | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. | [optional] 
**lun** | **int** | Optional: FC target lun number | [optional] 
**read_only** | **bool** | Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | [optional] 
**target_wwns** | **List[str]** | Optional: FC target worldwide names (WWNs) | [optional] 
**wwids** | **List[str]** | Optional: FC volume world wide identifiers (wwids) Either wwids or combination of targetWWNs and lun must be set, but not both simultaneously. | [optional] 

## Example

```python
from argo_workflows.models.fc_volume_source import FCVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of FCVolumeSource from a JSON string
fc_volume_source_instance = FCVolumeSource.from_json(json)
# print the JSON string representation of the object
print(FCVolumeSource.to_json())

# convert the object into a dict
fc_volume_source_dict = fc_volume_source_instance.to_dict()
# create an instance of FCVolumeSource from a dict
fc_volume_source_form_dict = fc_volume_source.from_dict(fc_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


