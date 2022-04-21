# IoArgoprojWorkflowV1alpha1WebHDFSArtifact


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_type** | **str** |  | [optional] 
**client_cert** | [**IoArgoprojWorkflowV1alpha1ClientCertAuth**](IoArgoprojWorkflowV1alpha1ClientCertAuth.md) |  | [optional] 
**endpoint** | **str** | webHDFS endpoint | [optional] 
**headers** | [**[IoArgoprojWorkflowV1alpha1Header]**](IoArgoprojWorkflowV1alpha1Header.md) | Headers are an optional list of headers to send with HTTP requests for artifacts | [optional] 
**oauth2** | [**IoArgoprojWorkflowV1alpha1OAuth2Auth**](IoArgoprojWorkflowV1alpha1OAuth2Auth.md) |  | [optional] 
**overwrite** | **bool** | whether to overwrite existing output artifacts (default: unset, meaning the endpoint&#39;s default behavior is used) | [optional] 
**path** | **str** | path to the artifact | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


