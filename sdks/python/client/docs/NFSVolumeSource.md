# NFSVolumeSource

Represents an NFS mount that lasts the lifetime of a pod. NFS volumes do not support ownership management or SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**path** | **str** | Path that is exported by the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs | 
**read_only** | **bool** | ReadOnly here will force the NFS export to be mounted with read-only permissions. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs | [optional] 
**server** | **str** | Server is the hostname or IP address of the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs | 

## Example

```python
from argo_workflows.models.nfs_volume_source import NFSVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of NFSVolumeSource from a JSON string
nfs_volume_source_instance = NFSVolumeSource.from_json(json)
# print the JSON string representation of the object
print(NFSVolumeSource.to_json())

# convert the object into a dict
nfs_volume_source_dict = nfs_volume_source_instance.to_dict()
# create an instance of NFSVolumeSource from a dict
nfs_volume_source_form_dict = nfs_volume_source.from_dict(nfs_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


