

# IoArgoprojWorkflowV1alpha1AzureBlobArtifactRepository

AzureBlobArtifactRepository defines the controller configuration for an artifactory artifact repository

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**accountKeySecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**blobNameFormat** | **String** | BlobNameFormat is defines the format of how to store blob names. Can reference workflow variables |  [optional]
**container** | **String** | Container is the container where resources will be stored | 
**endpoint** | **String** | Endpoint is the service url associated with an account. It is most likely \&quot;https://&lt;ACCOUNT_NAME&gt;.blob.core.windows.net\&quot; | 
**useSDKCreds** | **Boolean** | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. |  [optional]



