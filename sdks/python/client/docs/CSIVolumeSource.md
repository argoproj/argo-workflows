# CSIVolumeSource

Represents a source location of a volume to mount, managed by an external CSI driver

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**driver** | **str** | Driver is the name of the CSI driver that handles this volume. Consult with your admin for the correct name as registered in the cluster. | 
**fs_type** | **str** | Filesystem type to mount. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. If not provided, the empty value is passed to the associated CSI driver which will determine the default filesystem to apply. | [optional] 
**node_publish_secret_ref** | [**LocalObjectReference**](LocalObjectReference.md) |  | [optional] 
**read_only** | **bool** | Specifies a read-only configuration for the volume. Defaults to false (read/write). | [optional] 
**volume_attributes** | **Dict[str, str]** | VolumeAttributes stores driver-specific properties that are passed to the CSI driver. Consult your driver&#39;s documentation for supported values. | [optional] 

## Example

```python
from argo_workflows.models.csi_volume_source import CSIVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of CSIVolumeSource from a JSON string
csi_volume_source_instance = CSIVolumeSource.from_json(json)
# print the JSON string representation of the object
print(CSIVolumeSource.to_json())

# convert the object into a dict
csi_volume_source_dict = csi_volume_source_instance.to_dict()
# create an instance of CSIVolumeSource from a dict
csi_volume_source_form_dict = csi_volume_source.from_dict(csi_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


