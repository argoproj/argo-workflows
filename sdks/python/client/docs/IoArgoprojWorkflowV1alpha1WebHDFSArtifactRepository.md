# IoArgoprojWorkflowV1alpha1WebHDFSArtifactRepository

WebHDFSArtifactRepository defines the controller configuration for a webHDFS artifact repository

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_type** | **str** |  | [optional] 
**client_cert** | [**IoArgoprojWorkflowV1alpha1ClientCertAuth**](IoArgoprojWorkflowV1alpha1ClientCertAuth.md) |  | [optional] 
**endpoint** | **str** |  | [optional] 
**headers** | [**[IoArgoprojWorkflowV1alpha1Header]**](IoArgoprojWorkflowV1alpha1Header.md) | Optional headers to be passed in the webHDFS HTTP requests | [optional] 
**oauth2** | [**IoArgoprojWorkflowV1alpha1OAuth2Auth**](IoArgoprojWorkflowV1alpha1OAuth2Auth.md) |  | [optional] 
**overwrite** | **bool** | whether to overwrite existing files | [optional] 
**path_format** | **str** | PathFormat is defines the format of path to store a file. Can reference workflow variables | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


