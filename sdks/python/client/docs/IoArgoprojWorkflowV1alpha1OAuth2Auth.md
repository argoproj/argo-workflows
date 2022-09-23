# IoArgoprojWorkflowV1alpha1OAuth2Auth

OAuth2Auth holds all information for client authentication via OAuth2 tokens

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**client_id_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**client_secret_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**endpoint_params** | [**[IoArgoprojWorkflowV1alpha1OAuth2EndpointParam]**](IoArgoprojWorkflowV1alpha1OAuth2EndpointParam.md) |  | [optional] 
**scopes** | **[str]** |  | [optional] 
**token_url_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


