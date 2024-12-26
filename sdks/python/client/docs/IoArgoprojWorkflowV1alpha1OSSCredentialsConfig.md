# IoArgoprojWorkflowV1alpha1OSSCredentialsConfig

OSSCredentialsConfig specifies the credential configuration for OSS

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**o_idc_provider_arn** | **str** | OidcProviderARN is the Alibaba Cloud Resource Name (ARN) of the OIDC IdP. | [optional] 
**o_idc_token_file_path** | **str** | OidcTokenFile is the file path of the OIDC token. | [optional] 
**role_arn** | **str** | RoleARN is the Alibaba Cloud Resource Name(ARN) of the role to assume. | [optional] 
**role_session_name** | **str** | RoleSessionName is the session name of the role to assume. | [optional] 
**s_ts_endpoint** | **str** | STSEndpoint is the endpoint of the STS service. | [optional] 
**type** | **str** | Type specifies the credential type. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


