# IoArgoprojWorkflowV1alpha1ArchiveStrategy

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**none** | Option<[**serde_json::Value**](.md)> | NoneStrategy indicates to skip tar process and upload the files or directory tree as independent files. Note that if the artifact is a directory, the artifact driver must support the ability to save/load the directory appropriately. | [optional]
**tar** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1TarStrategy**](io.argoproj.workflow.v1alpha1.TarStrategy.md)> |  | [optional]
**zip** | Option<[**serde_json::Value**](.md)> | ZipStrategy will unzip zipped input artifacts | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


