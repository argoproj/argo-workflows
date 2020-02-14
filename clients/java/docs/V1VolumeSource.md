

# V1VolumeSource

Represents the source of a volume to mount. Only one of its members may be specified.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**awsElasticBlockStore** | [**V1AWSElasticBlockStoreVolumeSource**](V1AWSElasticBlockStoreVolumeSource.md) |  |  [optional]
**azureDisk** | [**V1AzureDiskVolumeSource**](V1AzureDiskVolumeSource.md) |  |  [optional]
**azureFile** | [**V1AzureFileVolumeSource**](V1AzureFileVolumeSource.md) |  |  [optional]
**cephfs** | [**V1CephFSVolumeSource**](V1CephFSVolumeSource.md) |  |  [optional]
**cinder** | [**V1CinderVolumeSource**](V1CinderVolumeSource.md) |  |  [optional]
**configMap** | [**V1ConfigMapVolumeSource**](V1ConfigMapVolumeSource.md) |  |  [optional]
**csi** | [**V1CSIVolumeSource**](V1CSIVolumeSource.md) |  |  [optional]
**downwardAPI** | [**V1DownwardAPIVolumeSource**](V1DownwardAPIVolumeSource.md) |  |  [optional]
**emptyDir** | [**V1EmptyDirVolumeSource**](V1EmptyDirVolumeSource.md) |  |  [optional]
**fc** | [**V1FCVolumeSource**](V1FCVolumeSource.md) |  |  [optional]
**flexVolume** | [**V1FlexVolumeSource**](V1FlexVolumeSource.md) |  |  [optional]
**flocker** | [**V1FlockerVolumeSource**](V1FlockerVolumeSource.md) |  |  [optional]
**gcePersistentDisk** | [**V1GCEPersistentDiskVolumeSource**](V1GCEPersistentDiskVolumeSource.md) |  |  [optional]
**gitRepo** | [**V1GitRepoVolumeSource**](V1GitRepoVolumeSource.md) |  |  [optional]
**glusterfs** | [**V1GlusterfsVolumeSource**](V1GlusterfsVolumeSource.md) |  |  [optional]
**hostPath** | [**V1HostPathVolumeSource**](V1HostPathVolumeSource.md) |  |  [optional]
**iscsi** | [**V1ISCSIVolumeSource**](V1ISCSIVolumeSource.md) |  |  [optional]
**nfs** | [**V1NFSVolumeSource**](V1NFSVolumeSource.md) |  |  [optional]
**persistentVolumeClaim** | [**V1PersistentVolumeClaimVolumeSource**](V1PersistentVolumeClaimVolumeSource.md) |  |  [optional]
**photonPersistentDisk** | [**V1PhotonPersistentDiskVolumeSource**](V1PhotonPersistentDiskVolumeSource.md) |  |  [optional]
**portworxVolume** | [**V1PortworxVolumeSource**](V1PortworxVolumeSource.md) |  |  [optional]
**projected** | [**V1ProjectedVolumeSource**](V1ProjectedVolumeSource.md) |  |  [optional]
**quobyte** | [**V1QuobyteVolumeSource**](V1QuobyteVolumeSource.md) |  |  [optional]
**rbd** | [**V1RBDVolumeSource**](V1RBDVolumeSource.md) |  |  [optional]
**scaleIO** | [**V1ScaleIOVolumeSource**](V1ScaleIOVolumeSource.md) |  |  [optional]
**secret** | [**V1SecretVolumeSource**](V1SecretVolumeSource.md) |  |  [optional]
**storageos** | [**V1StorageOSVolumeSource**](V1StorageOSVolumeSource.md) |  |  [optional]
**vsphereVolume** | [**V1VsphereVirtualDiskVolumeSource**](V1VsphereVirtualDiskVolumeSource.md) |  |  [optional]



