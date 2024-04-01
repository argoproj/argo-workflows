# AzureFileVolumeSource

AzureFile represents an Azure File Service mount on the host and bind mount to the pod.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**read_only** | **bool** | Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. | [optional] 
**secret_name** | **str** | the name of secret that contains Azure Storage Account Name and Key | 
**share_name** | **str** | Share Name | 

## Example

```python
from argo_workflows.models.azure_file_volume_source import AzureFileVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of AzureFileVolumeSource from a JSON string
azure_file_volume_source_instance = AzureFileVolumeSource.from_json(json)
# print the JSON string representation of the object
print(AzureFileVolumeSource.to_json())

# convert the object into a dict
azure_file_volume_source_dict = azure_file_volume_source_instance.to_dict()
# create an instance of AzureFileVolumeSource from a dict
azure_file_volume_source_form_dict = azure_file_volume_source.from_dict(azure_file_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


