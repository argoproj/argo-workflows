# GitRepoVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**directory** | Option<**String**> | Target directory name. Must not contain or start with '..'.  If '.' is supplied, the volume directory will be the git repository.  Otherwise, if specified, the volume will contain the git repository in the subdirectory with the given name. | [optional]
**repository** | **String** | Repository URL | 
**revision** | Option<**String**> | Commit hash for the specified revision. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


