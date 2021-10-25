# V1alpha1ArchiveStrategy

ArchiveStrategy describes how to archive files/directory when saving artifacts
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**_none** | [**object**](.md) | NoneStrategy indicates to skip tar process and upload the files or directory tree as independent files. Note that if the artifact is a directory, the artifact driver must support the ability to save/load the directory appropriately. | [optional] 
**tar** | [**V1alpha1TarStrategy**](V1alpha1TarStrategy.md) |  | [optional] 
**zip** | [**object**](.md) | ZipStrategy will unzip zipped input artifacts | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


