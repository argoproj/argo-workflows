

# IoArgoprojWorkflowV1alpha1HTTPArtifact

HTTPArtifact allows a file served on HTTP to be placed as an input artifact in a container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**basicAuth** | [**IoArgoprojWorkflowV1alpha1BasicAuth**](IoArgoprojWorkflowV1alpha1BasicAuth.md) |  |  [optional]
**clientCert** | [**IoArgoprojWorkflowV1alpha1ClientCertAuth**](IoArgoprojWorkflowV1alpha1ClientCertAuth.md) |  |  [optional]
**followTemporaryRedirects** | **Boolean** | whether to follow temporary redirects, needed for webHDFS |  [optional]
**headers** | [**List&lt;IoArgoprojWorkflowV1alpha1Header&gt;**](IoArgoprojWorkflowV1alpha1Header.md) | Headers are an optional list of headers to send with HTTP requests for artifacts |  [optional]
**oauth2** | [**IoArgoprojWorkflowV1alpha1OAuth2Auth**](IoArgoprojWorkflowV1alpha1OAuth2Auth.md) |  |  [optional]
**url** | **String** | URL of the artifact | 



