# IoArgoprojWorkflowV1alpha1ArchiveStrategy

ArchiveStrategy describes how to archive files/directory when saving artifacts

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**_none** | **bool, date, datetime, dict, float, int, list, str, none_type** | NoneStrategy indicates to skip tar process and upload the files or directory tree as independent files. Note that if the artifact is a directory, the artifact driver must support the ability to save/load the directory appropriately. | [optional] 
**tar** | [**IoArgoprojWorkflowV1alpha1TarStrategy**](IoArgoprojWorkflowV1alpha1TarStrategy.md) |  | [optional] 
**zip** | **bool, date, datetime, dict, float, int, list, str, none_type** | ZipStrategy will unzip zipped input artifacts | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


