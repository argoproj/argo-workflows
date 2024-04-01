# CinderVolumeSource

Represents a cinder volume resource in Openstack. A Cinder volume must exist before mounting to a container. The volume must also be in the same region as the kubelet. Cinder volumes support ownership management and SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | **str** | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://examples.k8s.io/mysql-cinder-pd/README.md | [optional] 
**read_only** | **bool** | Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/mysql-cinder-pd/README.md | [optional] 
**secret_ref** | [**LocalObjectReference**](LocalObjectReference.md) |  | [optional] 
**volume_id** | **str** | volume id used to identify the volume in cinder. More info: https://examples.k8s.io/mysql-cinder-pd/README.md | 

## Example

```python
from argo_workflows.models.cinder_volume_source import CinderVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of CinderVolumeSource from a JSON string
cinder_volume_source_instance = CinderVolumeSource.from_json(json)
# print the JSON string representation of the object
print(CinderVolumeSource.to_json())

# convert the object into a dict
cinder_volume_source_dict = cinder_volume_source_instance.to_dict()
# create an instance of CinderVolumeSource from a dict
cinder_volume_source_form_dict = cinder_volume_source.from_dict(cinder_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


