# RBDVolumeSource

Represents a Rados Block Device mount that lasts the lifetime of a pod. RBD volumes support ownership management and SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | **str** | Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#rbd | [optional] 
**image** | **str** | The rados image name. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it | 
**keyring** | **str** | Keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it | [optional] 
**monitors** | **List[str]** | A collection of Ceph monitors. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it | 
**pool** | **str** | The rados pool name. Default is rbd. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it | [optional] 
**read_only** | **bool** | ReadOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it | [optional] 
**secret_ref** | [**LocalObjectReference**](LocalObjectReference.md) |  | [optional] 
**user** | **str** | The rados user name. Default is admin. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it | [optional] 

## Example

```python
from argo_workflows.models.rbd_volume_source import RBDVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of RBDVolumeSource from a JSON string
rbd_volume_source_instance = RBDVolumeSource.from_json(json)
# print the JSON string representation of the object
print(RBDVolumeSource.to_json())

# convert the object into a dict
rbd_volume_source_dict = rbd_volume_source_instance.to_dict()
# create an instance of RBDVolumeSource from a dict
rbd_volume_source_form_dict = rbd_volume_source.from_dict(rbd_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


