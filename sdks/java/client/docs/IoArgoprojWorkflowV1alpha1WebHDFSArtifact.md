

# IoArgoprojWorkflowV1alpha1WebHDFSArtifact


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**authType** | **String** |  |  [optional]
**clientCert** | [**IoArgoprojWorkflowV1alpha1ClientCertAuth**](IoArgoprojWorkflowV1alpha1ClientCertAuth.md) |  |  [optional]
**endpoint** | **String** | webHDFS endpoint |  [optional]
**headers** | [**List&lt;IoArgoprojWorkflowV1alpha1Header&gt;**](IoArgoprojWorkflowV1alpha1Header.md) | Headers are an optional list of headers to send with HTTP requests for artifacts |  [optional]
**oauth2** | [**IoArgoprojWorkflowV1alpha1OAuth2Auth**](IoArgoprojWorkflowV1alpha1OAuth2Auth.md) |  |  [optional]
**overwrite** | **Boolean** | whether to overwrite existing output artifacts (default: unset, meaning the endpoint&#39;s default behavior is used) |  [optional]
**path** | **String** | path to the artifact |  [optional]



