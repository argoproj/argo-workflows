

# IoArgoprojWorkflowV1alpha1HTTPArtifact

HTTPArtifact allows a file served on HTTP to be placed as an input artifact in a container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**IoArgoprojWorkflowV1alpha1HTTPAuth**](IoArgoprojWorkflowV1alpha1HTTPAuth.md) |  |  [optional]
**headers** | [**List&lt;IoArgoprojWorkflowV1alpha1Header&gt;**](IoArgoprojWorkflowV1alpha1Header.md) | Headers are an optional list of headers to send with HTTP requests for artifacts |  [optional]
**saveStreamViaFile** | **Boolean** | SaveStreamViaFile buffers a streamed upload to a temporary file before sending it, so a 307/308 redirect (e.g. webHDFS) can be followed by re-sending the body. When false (the default) SaveStream sends the reader directly and cannot follow such a redirect, since a one-shot reader cannot be replayed. |  [optional]
**url** | **String** | URL of the artifact | 



