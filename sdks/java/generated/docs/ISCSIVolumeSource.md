

# ISCSIVolumeSource

Represents an ISCSI disk. ISCSI volumes can only be mounted as read/write once. ISCSI volumes support ownership management and SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**chapAuthDiscovery** | **Boolean** | whether support iSCSI Discovery CHAP authentication |  [optional]
**chapAuthSession** | **Boolean** | whether support iSCSI Session CHAP authentication |  [optional]
**fsType** | **String** | Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi |  [optional]
**initiatorName** | **String** | Custom iSCSI Initiator Name. If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface &lt;target portal&gt;:&lt;volume name&gt; will be created for the connection. |  [optional]
**iqn** | **String** | Target iSCSI Qualified Name. | 
**iscsiInterface** | **String** | iSCSI Interface Name that uses an iSCSI transport. Defaults to &#39;default&#39; (tcp). |  [optional]
**lun** | **Integer** | iSCSI Target Lun number. | 
**portals** | **List&lt;String&gt;** | iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). |  [optional]
**readOnly** | **Boolean** | ReadOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. |  [optional]
**secretRef** | [**io.kubernetes.client.openapi.models.V1LocalObjectReference**](io.kubernetes.client.openapi.models.V1LocalObjectReference.md) |  |  [optional]
**targetPortal** | **String** | iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). | 



