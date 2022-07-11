# IoArgoprojWorkflowV1alpha1AzureArtifact

AzureArtifact is the location of a an Azure Storage artifact

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**blob** | **str** | Blob is the blob name (i.e., path) in the container where the artifact resides | 
**container** | **str** | Container is the container where resources will be stored | 
**endpoint** | **str** | Endpoint is the service url associated with an account. It is most likely \&quot;https://&lt;ACCOUNT_NAME&gt;.blob.core.windows.net\&quot; | 
**account_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**use_sdk_creds** | **bool** | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


