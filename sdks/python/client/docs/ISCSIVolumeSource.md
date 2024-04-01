# ISCSIVolumeSource

Represents an ISCSI disk. ISCSI volumes can only be mounted as read/write once. ISCSI volumes support ownership management and SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**chap_auth_discovery** | **bool** | whether support iSCSI Discovery CHAP authentication | [optional] 
**chap_auth_session** | **bool** | whether support iSCSI Session CHAP authentication | [optional] 
**fs_type** | **str** | Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi | [optional] 
**initiator_name** | **str** | Custom iSCSI Initiator Name. If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface &lt;target portal&gt;:&lt;volume name&gt; will be created for the connection. | [optional] 
**iqn** | **str** | Target iSCSI Qualified Name. | 
**iscsi_interface** | **str** | iSCSI Interface Name that uses an iSCSI transport. Defaults to &#39;default&#39; (tcp). | [optional] 
**lun** | **int** | iSCSI Target Lun number. | 
**portals** | **List[str]** | iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). | [optional] 
**read_only** | **bool** | ReadOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. | [optional] 
**secret_ref** | [**LocalObjectReference**](LocalObjectReference.md) |  | [optional] 
**target_portal** | **str** | iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). | 

## Example

```python
from argo_workflows.models.iscsi_volume_source import ISCSIVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of ISCSIVolumeSource from a JSON string
iscsi_volume_source_instance = ISCSIVolumeSource.from_json(json)
# print the JSON string representation of the object
print(ISCSIVolumeSource.to_json())

# convert the object into a dict
iscsi_volume_source_dict = iscsi_volume_source_instance.to_dict()
# create an instance of ISCSIVolumeSource from a dict
iscsi_volume_source_form_dict = iscsi_volume_source.from_dict(iscsi_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


