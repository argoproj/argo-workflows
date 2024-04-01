# GlusterfsVolumeSource

Represents a Glusterfs mount that lasts the lifetime of a pod. Glusterfs volumes do not support ownership management or SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**endpoints** | **str** | EndpointsName is the endpoint name that details Glusterfs topology. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod | 
**path** | **str** | Path is the Glusterfs volume path. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod | 
**read_only** | **bool** | ReadOnly here will force the Glusterfs volume to be mounted with read-only permissions. Defaults to false. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod | [optional] 

## Example

```python
from argo_workflows.models.glusterfs_volume_source import GlusterfsVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of GlusterfsVolumeSource from a JSON string
glusterfs_volume_source_instance = GlusterfsVolumeSource.from_json(json)
# print the JSON string representation of the object
print(GlusterfsVolumeSource.to_json())

# convert the object into a dict
glusterfs_volume_source_dict = glusterfs_volume_source_instance.to_dict()
# create an instance of GlusterfsVolumeSource from a dict
glusterfs_volume_source_form_dict = glusterfs_volume_source.from_dict(glusterfs_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


