# ProjectedVolumeSource

Represents a projected volume source

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**default_mode** | **int** | Mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set. | [optional] 
**sources** | [**List[VolumeProjection]**](VolumeProjection.md) | list of volume projections | [optional] 

## Example

```python
from argo_workflows.models.projected_volume_source import ProjectedVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of ProjectedVolumeSource from a JSON string
projected_volume_source_instance = ProjectedVolumeSource.from_json(json)
# print the JSON string representation of the object
print(ProjectedVolumeSource.to_json())

# convert the object into a dict
projected_volume_source_dict = projected_volume_source_instance.to_dict()
# create an instance of ProjectedVolumeSource from a dict
projected_volume_source_form_dict = projected_volume_source.from_dict(projected_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


