

# Volume

Volume represents a named volume in a pod that may be accessed by any container in the pod.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**awsElasticBlockStore** | [**AWSElasticBlockStoreVolumeSource**](AWSElasticBlockStoreVolumeSource.md) |  |  [optional]
**azureDisk** | [**AzureDiskVolumeSource**](AzureDiskVolumeSource.md) |  |  [optional]
**azureFile** | [**AzureFileVolumeSource**](AzureFileVolumeSource.md) |  |  [optional]
**cephfs** | [**CephFSVolumeSource**](CephFSVolumeSource.md) |  |  [optional]
**cinder** | [**CinderVolumeSource**](CinderVolumeSource.md) |  |  [optional]
**configMap** | [**ConfigMapVolumeSource**](ConfigMapVolumeSource.md) |  |  [optional]
**csi** | [**CSIVolumeSource**](CSIVolumeSource.md) |  |  [optional]
**downwardAPI** | [**DownwardAPIVolumeSource**](DownwardAPIVolumeSource.md) |  |  [optional]
**emptyDir** | [**EmptyDirVolumeSource**](EmptyDirVolumeSource.md) |  |  [optional]
**ephemeral** | [**EphemeralVolumeSource**](EphemeralVolumeSource.md) |  |  [optional]
**fc** | [**FCVolumeSource**](FCVolumeSource.md) |  |  [optional]
**flexVolume** | [**FlexVolumeSource**](FlexVolumeSource.md) |  |  [optional]
**flocker** | [**FlockerVolumeSource**](FlockerVolumeSource.md) |  |  [optional]
**gcePersistentDisk** | [**GCEPersistentDiskVolumeSource**](GCEPersistentDiskVolumeSource.md) |  |  [optional]
**gitRepo** | [**GitRepoVolumeSource**](GitRepoVolumeSource.md) |  |  [optional]
**glusterfs** | [**GlusterfsVolumeSource**](GlusterfsVolumeSource.md) |  |  [optional]
**hostPath** | [**HostPathVolumeSource**](HostPathVolumeSource.md) |  |  [optional]
**iscsi** | [**ISCSIVolumeSource**](ISCSIVolumeSource.md) |  |  [optional]
**name** | **String** | Volume&#39;s name. Must be a DNS_LABEL and unique within the pod. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names | 
**nfs** | [**NFSVolumeSource**](NFSVolumeSource.md) |  |  [optional]
**persistentVolumeClaim** | [**PersistentVolumeClaimVolumeSource**](PersistentVolumeClaimVolumeSource.md) |  |  [optional]
**photonPersistentDisk** | [**PhotonPersistentDiskVolumeSource**](PhotonPersistentDiskVolumeSource.md) |  |  [optional]
**portworxVolume** | [**PortworxVolumeSource**](PortworxVolumeSource.md) |  |  [optional]
**projected** | [**ProjectedVolumeSource**](ProjectedVolumeSource.md) |  |  [optional]
**quobyte** | [**QuobyteVolumeSource**](QuobyteVolumeSource.md) |  |  [optional]
**rbd** | [**RBDVolumeSource**](RBDVolumeSource.md) |  |  [optional]
**scaleIO** | [**ScaleIOVolumeSource**](ScaleIOVolumeSource.md) |  |  [optional]
**secret** | [**SecretVolumeSource**](SecretVolumeSource.md) |  |  [optional]
**storageos** | [**StorageOSVolumeSource**](StorageOSVolumeSource.md) |  |  [optional]
**vsphereVolume** | [**VsphereVirtualDiskVolumeSource**](VsphereVirtualDiskVolumeSource.md) |  |  [optional]



