# IoArgoprojWorkflowV1alpha1AzureArtifact

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**account_key_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**blob** | **String** | Blob is the blob name (i.e., path) in the container where the artifact resides | 
**container** | **String** | Container is the container where resources will be stored | 
**endpoint** | **String** | Endpoint is the service url associated with an account. It is most likely \"https://<ACCOUNT_NAME>.blob.core.windows.net\" | 
**use_sdk_creds** | Option<**bool**> | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


