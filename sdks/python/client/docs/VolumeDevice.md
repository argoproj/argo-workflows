# VolumeDevice

volumeDevice describes a mapping of a raw block device within a container.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_path** | **str** | devicePath is the path inside of the container that the device will be mapped to. | 
**name** | **str** | name must match the name of a persistentVolumeClaim in the pod | 

## Example

```python
from argo_workflows.models.volume_device import VolumeDevice

# TODO update the JSON string below
json = "{}"
# create an instance of VolumeDevice from a JSON string
volume_device_instance = VolumeDevice.from_json(json)
# print the JSON string representation of the object
print(VolumeDevice.to_json())

# convert the object into a dict
volume_device_dict = volume_device_instance.to_dict()
# create an instance of VolumeDevice from a dict
volume_device_form_dict = volume_device.from_dict(volume_device_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


