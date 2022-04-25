# IoArgoprojWorkflowV1alpha1HTTPArtifact

HTTPArtifact allows a file served on HTTP to be placed as an input artifact in a container

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**url** | **str** | URL of the artifact | 
**basic_auth** | [**IoArgoprojWorkflowV1alpha1BasicAuth**](IoArgoprojWorkflowV1alpha1BasicAuth.md) |  | [optional] 
**client_cert** | [**IoArgoprojWorkflowV1alpha1ClientCertAuth**](IoArgoprojWorkflowV1alpha1ClientCertAuth.md) |  | [optional] 
**follow_temporary_redirects** | **bool** | whether to follow temporary redirects, needed for webHDFS | [optional] 
**headers** | [**[IoArgoprojWorkflowV1alpha1Header]**](IoArgoprojWorkflowV1alpha1Header.md) | Headers are an optional list of headers to send with HTTP requests for artifacts | [optional] 
**oauth2** | [**IoArgoprojWorkflowV1alpha1OAuth2Auth**](IoArgoprojWorkflowV1alpha1OAuth2Auth.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


