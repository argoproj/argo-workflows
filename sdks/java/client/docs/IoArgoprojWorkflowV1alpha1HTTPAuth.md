

# IoArgoprojWorkflowV1alpha1HTTPAuth

HTTPAuth describes how to authenticate to an HTTP artifact server. When more than one method is set, the Authorization header comes from OAuth2 if set, otherwise from Bearer, otherwise from BasicAuth. ClientCert replaces the HTTP client, so setting it disables OAuth2; it combines with a Bearer or BasicAuth header.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**basicAuth** | [**IoArgoprojWorkflowV1alpha1BasicAuth**](IoArgoprojWorkflowV1alpha1BasicAuth.md) |  |  [optional]
**bearer** | [**IoArgoprojWorkflowV1alpha1BearerAuth**](IoArgoprojWorkflowV1alpha1BearerAuth.md) |  |  [optional]
**clientCert** | [**IoArgoprojWorkflowV1alpha1ClientCertAuth**](IoArgoprojWorkflowV1alpha1ClientCertAuth.md) |  |  [optional]
**oauth2** | [**IoArgoprojWorkflowV1alpha1OAuth2Auth**](IoArgoprojWorkflowV1alpha1OAuth2Auth.md) |  |  [optional]



