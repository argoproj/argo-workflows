# DownwardAPIProjection

Represents downward API info for projecting into a projected volume. Note that this is identical to a downwardAPI volume source without the default mode.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[DownwardAPIVolumeFile]**](DownwardAPIVolumeFile.md) | Items is a list of DownwardAPIVolume file | [optional] 

## Example

```python
from argo_workflows.models.downward_api_projection import DownwardAPIProjection

# TODO update the JSON string below
json = "{}"
# create an instance of DownwardAPIProjection from a JSON string
downward_api_projection_instance = DownwardAPIProjection.from_json(json)
# print the JSON string representation of the object
print(DownwardAPIProjection.to_json())

# convert the object into a dict
downward_api_projection_dict = downward_api_projection_instance.to_dict()
# create an instance of DownwardAPIProjection from a dict
downward_api_projection_form_dict = downward_api_projection.from_dict(downward_api_projection_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


