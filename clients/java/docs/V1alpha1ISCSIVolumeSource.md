

# V1alpha1ISCSIVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**chapAuthDiscovery** | **Boolean** |  |  [optional]
**chapAuthSession** | **Boolean** |  |  [optional]
**fsType** | **String** |  |  [optional]
**initiatorName** | **String** |  |  [optional]
**iqn** | **String** | Target iSCSI Qualified Name. |  [optional]
**iscsiInterface** | **String** |  |  [optional]
**lun** | **Integer** | iSCSI Target Lun number. |  [optional]
**portals** | **List&lt;String&gt;** |  |  [optional]
**readOnly** | **Boolean** |  |  [optional]
**secretRef** | [**V1alpha1LocalObjectReference**](V1alpha1LocalObjectReference.md) |  |  [optional]
**targetPortal** | **String** | iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). |  [optional]



