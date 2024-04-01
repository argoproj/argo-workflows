# CephFSVolumeSource

Represents a Ceph Filesystem mount that lasts the lifetime of a pod Cephfs volumes do not support ownership management or SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**monitors** | **List[str]** | Required: Monitors is a collection of Ceph monitors More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it | 
**path** | **str** | Optional: Used as the mounted root, rather than the full Ceph tree, default is / | [optional] 
**read_only** | **bool** | Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it | [optional] 
**secret_file** | **str** | Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it | [optional] 
**secret_ref** | [**LocalObjectReference**](LocalObjectReference.md) |  | [optional] 
**user** | **str** | Optional: User is the rados user name, default is admin More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it | [optional] 

## Example

```python
from argo_workflows.models.ceph_fs_volume_source import CephFSVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of CephFSVolumeSource from a JSON string
ceph_fs_volume_source_instance = CephFSVolumeSource.from_json(json)
# print the JSON string representation of the object
print(CephFSVolumeSource.to_json())

# convert the object into a dict
ceph_fs_volume_source_dict = ceph_fs_volume_source_instance.to_dict()
# create an instance of CephFSVolumeSource from a dict
ceph_fs_volume_source_form_dict = ceph_fs_volume_source.from_dict(ceph_fs_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


