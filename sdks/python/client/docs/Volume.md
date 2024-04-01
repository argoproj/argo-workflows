# Volume

Volume represents a named volume in a pod that may be accessed by any container in the pod.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
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
**iscsi** | [**ISCSIVolumeSource**](ISCSIVolumeSource.md) |  | [optional] 
**name** | **str** | Volume&#39;s name. Must be a DNS_LABEL and unique within the pod. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names | 
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

## Example

```python
from argo_workflows.models.volume import Volume

# TODO update the JSON string below
json = "{}"
# create an instance of Volume from a JSON string
volume_instance = Volume.from_json(json)
# print the JSON string representation of the object
print(Volume.to_json())

# convert the object into a dict
volume_dict = volume_instance.to_dict()
# create an instance of Volume from a dict
volume_form_dict = volume.from_dict(volume_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


