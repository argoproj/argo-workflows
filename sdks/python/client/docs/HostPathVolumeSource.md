# HostPathVolumeSource

Represents a host path mapped into a pod. Host path volumes do not support ownership management or SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**path** | **str** | Path of the directory on the host. If the path is a symlink, it will follow the link to the real path. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath | 
**type** | **str** | Type for HostPath Volume Defaults to \&quot;\&quot; More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath | [optional] 

## Example

```python
from argo_workflows.models.host_path_volume_source import HostPathVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of HostPathVolumeSource from a JSON string
host_path_volume_source_instance = HostPathVolumeSource.from_json(json)
# print the JSON string representation of the object
print(HostPathVolumeSource.to_json())

# convert the object into a dict
host_path_volume_source_dict = host_path_volume_source_instance.to_dict()
# create an instance of HostPathVolumeSource from a dict
host_path_volume_source_form_dict = host_path_volume_source.from_dict(host_path_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


