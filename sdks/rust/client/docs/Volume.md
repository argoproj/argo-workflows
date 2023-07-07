# Volume

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**aws_elastic_block_store** | Option<[**crate::models::AwsElasticBlockStoreVolumeSource**](AWSElasticBlockStoreVolumeSource.md)> |  | [optional]
**azure_disk** | Option<[**crate::models::AzureDiskVolumeSource**](AzureDiskVolumeSource.md)> |  | [optional]
**azure_file** | Option<[**crate::models::AzureFileVolumeSource**](AzureFileVolumeSource.md)> |  | [optional]
**cephfs** | Option<[**crate::models::CephFsVolumeSource**](CephFSVolumeSource.md)> |  | [optional]
**cinder** | Option<[**crate::models::CinderVolumeSource**](CinderVolumeSource.md)> |  | [optional]
**config_map** | Option<[**crate::models::ConfigMapVolumeSource**](ConfigMapVolumeSource.md)> |  | [optional]
**csi** | Option<[**crate::models::CsiVolumeSource**](CSIVolumeSource.md)> |  | [optional]
**downward_api** | Option<[**crate::models::DownwardApiVolumeSource**](DownwardAPIVolumeSource.md)> |  | [optional]
**empty_dir** | Option<[**crate::models::EmptyDirVolumeSource**](EmptyDirVolumeSource.md)> |  | [optional]
**ephemeral** | Option<[**crate::models::EphemeralVolumeSource**](EphemeralVolumeSource.md)> |  | [optional]
**fc** | Option<[**crate::models::FcVolumeSource**](FCVolumeSource.md)> |  | [optional]
**flex_volume** | Option<[**crate::models::FlexVolumeSource**](FlexVolumeSource.md)> |  | [optional]
**flocker** | Option<[**crate::models::FlockerVolumeSource**](FlockerVolumeSource.md)> |  | [optional]
**gce_persistent_disk** | Option<[**crate::models::GcePersistentDiskVolumeSource**](GCEPersistentDiskVolumeSource.md)> |  | [optional]
**git_repo** | Option<[**crate::models::GitRepoVolumeSource**](GitRepoVolumeSource.md)> |  | [optional]
**glusterfs** | Option<[**crate::models::GlusterfsVolumeSource**](GlusterfsVolumeSource.md)> |  | [optional]
**host_path** | Option<[**crate::models::HostPathVolumeSource**](HostPathVolumeSource.md)> |  | [optional]
**iscsi** | Option<[**crate::models::IscsiVolumeSource**](ISCSIVolumeSource.md)> |  | [optional]
**name** | **String** | Volume's name. Must be a DNS_LABEL and unique within the pod. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names | 
**nfs** | Option<[**crate::models::NfsVolumeSource**](NFSVolumeSource.md)> |  | [optional]
**persistent_volume_claim** | Option<[**crate::models::PersistentVolumeClaimVolumeSource**](PersistentVolumeClaimVolumeSource.md)> |  | [optional]
**photon_persistent_disk** | Option<[**crate::models::PhotonPersistentDiskVolumeSource**](PhotonPersistentDiskVolumeSource.md)> |  | [optional]
**portworx_volume** | Option<[**crate::models::PortworxVolumeSource**](PortworxVolumeSource.md)> |  | [optional]
**projected** | Option<[**crate::models::ProjectedVolumeSource**](ProjectedVolumeSource.md)> |  | [optional]
**quobyte** | Option<[**crate::models::QuobyteVolumeSource**](QuobyteVolumeSource.md)> |  | [optional]
**rbd** | Option<[**crate::models::RbdVolumeSource**](RBDVolumeSource.md)> |  | [optional]
**scale_io** | Option<[**crate::models::ScaleIoVolumeSource**](ScaleIOVolumeSource.md)> |  | [optional]
**secret** | Option<[**crate::models::SecretVolumeSource**](SecretVolumeSource.md)> |  | [optional]
**storageos** | Option<[**crate::models::StorageOsVolumeSource**](StorageOSVolumeSource.md)> |  | [optional]
**vsphere_volume** | Option<[**crate::models::VsphereVirtualDiskVolumeSource**](VsphereVirtualDiskVolumeSource.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


