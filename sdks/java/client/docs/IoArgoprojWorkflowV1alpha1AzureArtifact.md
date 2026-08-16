

# IoArgoprojWorkflowV1alpha1AzureArtifact

AzureArtifact is the location of an Azure Storage artifact

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**accountKeySecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**blob** | **String** | Blob is the blob name (i.e., path) in the container where the artifact resides | 
**container** | **String** | Container is the container where resources will be stored | 
**endpoint** | **String** | Endpoint is the service url associated with an account. It is most likely \&quot;https://&lt;ACCOUNT_NAME&gt;.blob.core.windows.net\&quot; | 
**useSDKCreds** | **Boolean** | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. |  [optional]



