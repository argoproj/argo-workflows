# IscsiVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**chap_auth_discovery** | Option<**bool**> | whether support iSCSI Discovery CHAP authentication | [optional]
**chap_auth_session** | Option<**bool**> | whether support iSCSI Session CHAP authentication | [optional]
**fs_type** | Option<**String**> | Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi | [optional]
**initiator_name** | Option<**String**> | Custom iSCSI Initiator Name. If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface <target portal>:<volume name> will be created for the connection. | [optional]
**iqn** | **String** | Target iSCSI Qualified Name. | 
**iscsi_interface** | Option<**String**> | iSCSI Interface Name that uses an iSCSI transport. Defaults to 'default' (tcp). | [optional]
**lun** | **i32** | iSCSI Target Lun number. | 
**portals** | Option<**Vec<String>**> | iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). | [optional]
**read_only** | Option<**bool**> | ReadOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. | [optional]
**secret_ref** | Option<[**crate::models::LocalObjectReference**](LocalObjectReference.md)> |  | [optional]
**target_portal** | **String** | iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


