# FlexVolumeSource

FlexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**driver** | **str** | Driver is the name of the driver to use for this volume. | 
**fs_type** | **str** | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. The default filesystem depends on FlexVolume script. | [optional] 
**options** | **Dict[str, str]** | Optional: Extra command options if any. | [optional] 
**read_only** | **bool** | Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | [optional] 
**secret_ref** | [**LocalObjectReference**](LocalObjectReference.md) |  | [optional] 

## Example

```python
from argo_workflows.models.flex_volume_source import FlexVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of FlexVolumeSource from a JSON string
flex_volume_source_instance = FlexVolumeSource.from_json(json)
# print the JSON string representation of the object
print(FlexVolumeSource.to_json())

# convert the object into a dict
flex_volume_source_dict = flex_volume_source_instance.to_dict()
# create an instance of FlexVolumeSource from a dict
flex_volume_source_form_dict = flex_volume_source.from_dict(flex_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


