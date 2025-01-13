

# ISCSIVolumeSource

Represents an ISCSI disk. ISCSI volumes can only be mounted as read/write once. ISCSI volumes support ownership management and SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**chapAuthDiscovery** | **Boolean** | chapAuthDiscovery defines whether support iSCSI Discovery CHAP authentication |  [optional]
**chapAuthSession** | **Boolean** | chapAuthSession defines whether support iSCSI Session CHAP authentication |  [optional]
**fsType** | **String** | fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi |  [optional]
**initiatorName** | **String** | initiatorName is the custom iSCSI Initiator Name. If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface &lt;target portal&gt;:&lt;volume name&gt; will be created for the connection. |  [optional]
**iqn** | **String** | iqn is the target iSCSI Qualified Name. | 
**iscsiInterface** | **String** | iscsiInterface is the interface Name that uses an iSCSI transport. Defaults to &#39;default&#39; (tcp). |  [optional]
**lun** | **Integer** | lun represents iSCSI Target Lun number. | 
**portals** | **List&lt;String&gt;** | portals is the iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). |  [optional]
**readOnly** | **Boolean** | readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. |  [optional]
**secretRef** | [**io.kubernetes.client.openapi.models.V1LocalObjectReference**](io.kubernetes.client.openapi.models.V1LocalObjectReference.md) |  |  [optional]
**targetPortal** | **String** | targetPortal is iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). | 



