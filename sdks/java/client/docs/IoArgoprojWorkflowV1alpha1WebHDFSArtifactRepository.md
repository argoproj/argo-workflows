

# IoArgoprojWorkflowV1alpha1WebHDFSArtifactRepository

WebHDFSArtifactRepository defines the controller configuration for a webHDFS artifact repository

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**authType** | **String** |  |  [optional]
**clientCert** | [**IoArgoprojWorkflowV1alpha1ClientCertAuth**](IoArgoprojWorkflowV1alpha1ClientCertAuth.md) |  |  [optional]
**endpoint** | **String** |  |  [optional]
**headers** | [**List&lt;IoArgoprojWorkflowV1alpha1Header&gt;**](IoArgoprojWorkflowV1alpha1Header.md) | Optional headers to be passed in the webHDFS HTTP requests |  [optional]
**oauth2** | [**IoArgoprojWorkflowV1alpha1OAuth2Auth**](IoArgoprojWorkflowV1alpha1OAuth2Auth.md) |  |  [optional]
**overwrite** | **Boolean** | whether to overwrite existing files |  [optional]
**pathFormat** | **String** | PathFormat is defines the format of path to store a file. Can reference workflow variables |  [optional]



