# CinderVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | Option<**String**> | Filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: https://examples.k8s.io/mysql-cinder-pd/README.md | [optional]
**read_only** | Option<**bool**> | Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/mysql-cinder-pd/README.md | [optional]
**secret_ref** | Option<[**crate::models::LocalObjectReference**](LocalObjectReference.md)> |  | [optional]
**volume_id** | **String** | volume id used to identify the volume in cinder. More info: https://examples.k8s.io/mysql-cinder-pd/README.md | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


