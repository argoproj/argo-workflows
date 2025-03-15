# Volume

Volume represents a named volume in a pod that may be accessed by any container in the pod.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | name of the volume. Must be a DNS_LABEL and unique within the pod. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names | 
**aws_elastic_block_store** | [**AWSElasticBlockStoreVolumeSource**](AWSElasticBlockStoreVolumeSource.md) |  | [optional] 
**azure_disk** | [**AzureDiskVolumeSource**](AzureDiskVolumeSource.md) |  | [optional] 
**azure_file** | [**AzureFileVolumeSource**](AzureFileVolumeSource.md) |  | [optional] 
**cephfs** | [**CephFSVolumeSource**](CephFSVolumeSource.md) |  | [optional] 
**cinder** | [**CinderVolumeSource**](CinderVolumeSource.md) |  | [optional] 
**config_map** | [**ConfigMapVolumeSource**](ConfigMapVolumeSource.md) |  | [optional] 
**csi** | [**CSIVolumeSource**](CSIVolumeSource.md) |  | [optional] 
**downward_api** | [**DownwardAPIVolumeSource**](DownwardAPIVolumeSource.md) |  | [optional] 
**empty_dir** | [**EmptyDirVolumeSource**](EmptyDirVolumeSource.md) |  | [optional] 
**ephemeral** | [**EphemeralVolumeSource**](EphemeralVolumeSource.md) |  | [optional] 
**fc** | [**FCVolumeSource**](FCVolumeSource.md) |  | [optional] 
**flex_volume** | [**FlexVolumeSource**](FlexVolumeSource.md) |  | [optional] 
**flocker** | [**FlockerVolumeSource**](FlockerVolumeSource.md) |  | [optional] 
**gce_persistent_disk** | [**GCEPersistentDiskVolumeSource**](GCEPersistentDiskVolumeSource.md) |  | [optional] 
**git_repo** | [**GitRepoVolumeSource**](GitRepoVolumeSource.md) |  | [optional] 
**glusterfs** | [**GlusterfsVolumeSource**](GlusterfsVolumeSource.md) |  | [optional] 
**host_path** | [**HostPathVolumeSource**](HostPathVolumeSource.md) |  | [optional] 
**image** | [**ImageVolumeSource**](ImageVolumeSource.md) |  | [optional] 
**iscsi** | [**ISCSIVolumeSource**](ISCSIVolumeSource.md) |  | [optional] 
**nfs** | [**NFSVolumeSource**](NFSVolumeSource.md) |  | [optional] 
**persistent_volume_claim** | [**PersistentVolumeClaimVolumeSource**](PersistentVolumeClaimVolumeSource.md) |  | [optional] 
**photon_persistent_disk** | [**PhotonPersistentDiskVolumeSource**](PhotonPersistentDiskVolumeSource.md) |  | [optional] 
**portworx_volume** | [**PortworxVolumeSource**](PortworxVolumeSource.md) |  | [optional] 
**projected** | [**ProjectedVolumeSource**](ProjectedVolumeSource.md) |  | [optional] 
**quobyte** | [**QuobyteVolumeSource**](QuobyteVolumeSource.md) |  | [optional] 
**rbd** | [**RBDVolumeSource**](RBDVolumeSource.md) |  | [optional] 
**scale_io** | [**ScaleIOVolumeSource**](ScaleIOVolumeSource.md) |  | [optional] 
**secret** | [**SecretVolumeSource**](SecretVolumeSource.md) |  | [optional] 
**storageos** | [**StorageOSVolumeSource**](StorageOSVolumeSource.md) |  | [optional] 
**vsphere_volume** | [**VsphereVirtualDiskVolumeSource**](VsphereVirtualDiskVolumeSource.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


