# IoArgoprojWorkflowV1alpha1S3ArtifactRepository

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_key_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**bucket** | Option<**String**> | Bucket is the name of the bucket | [optional]
**create_bucket_if_not_present** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1CreateS3BucketOptions**](io.argoproj.workflow.v1alpha1.CreateS3BucketOptions.md)> |  | [optional]
**encryption_options** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1S3EncryptionOptions**](io.argoproj.workflow.v1alpha1.S3EncryptionOptions.md)> |  | [optional]
**endpoint** | Option<**String**> | Endpoint is the hostname of the bucket endpoint | [optional]
**insecure** | Option<**bool**> | Insecure will connect to the service with TLS | [optional]
**key_format** | Option<**String**> | KeyFormat is defines the format of how to store keys. Can reference workflow variables | [optional]
**key_prefix** | Option<**String**> | KeyPrefix is prefix used as part of the bucket key in which the controller will store artifacts. DEPRECATED. Use KeyFormat instead | [optional]
**region** | Option<**String**> | Region contains the optional bucket region | [optional]
**role_arn** | Option<**String**> | RoleARN is the Amazon Resource Name (ARN) of the role to assume. | [optional]
**secret_key_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**use_sdk_creds** | Option<**bool**> | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


