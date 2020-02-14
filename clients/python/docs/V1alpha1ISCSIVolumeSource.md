# V1alpha1ISCSIVolumeSource

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**chap_auth_discovery** | **bool** |  | [optional] 
**chap_auth_session** | **bool** |  | [optional] 
**fs_type** | **str** |  | [optional] 
**initiator_name** | **str** |  | [optional] 
**iqn** | **str** | Target iSCSI Qualified Name. | [optional] 
**iscsi_interface** | **str** |  | [optional] 
**lun** | **int** | iSCSI Target Lun number. | [optional] 
**portals** | **list[str]** |  | [optional] 
**read_only** | **bool** |  | [optional] 
**secret_ref** | [**V1alpha1LocalObjectReference**](V1alpha1LocalObjectReference.md) |  | [optional] 
**target_portal** | **str** | iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260). | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


