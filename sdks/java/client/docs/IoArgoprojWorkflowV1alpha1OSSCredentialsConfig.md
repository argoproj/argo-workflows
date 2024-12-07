

# IoArgoprojWorkflowV1alpha1OSSCredentialsConfig

OSSCredentialsConfig specifies the credential configuration for OSS

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**oIDCProviderArn** | **String** | OidcProviderARN is the Alibaba Cloud Resource Name (ARN) of the OIDC IdP. |  [optional]
**oIDCTokenFilePath** | **String** | OidcTokenFile is the file path of the OIDC token. |  [optional]
**roleArn** | **String** | RoleARN is the Alibaba Cloud Resource Name(ARN) of the role to assume. |  [optional]
**roleSessionName** | **String** | RoleSessionName is the session name of the role to assume. |  [optional]
**sTSEndpoint** | **String** | STSEndpoint is the endpoint of the STS service. |  [optional]
**type** | **String** | Type specifies the credential type. |  [optional]



