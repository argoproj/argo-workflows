# IoArgoprojWorkflowV1alpha1S3Artifact

S3Artifact is the location of an S3 artifact

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**bucket** | **str** | Bucket is the name of the bucket | [optional] 
**ca_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**create_bucket_if_not_present** | [**IoArgoprojWorkflowV1alpha1CreateS3BucketOptions**](IoArgoprojWorkflowV1alpha1CreateS3BucketOptions.md) |  | [optional] 
**encryption_options** | [**IoArgoprojWorkflowV1alpha1S3EncryptionOptions**](IoArgoprojWorkflowV1alpha1S3EncryptionOptions.md) |  | [optional] 
**endpoint** | **str** | Endpoint is the hostname of the bucket endpoint | [optional] 
**insecure** | **bool** | Insecure will connect to the service with TLS | [optional] 
**key** | **str** | Key is the key in the bucket where the artifact resides | [optional] 
**multipart_concurrency** | **int** | MultipartConcurrency is the number of concurrent multipart uploads. If not set, defaults to 4. | [optional] 
**multipart_part_size** | **int** | MultipartPartSize is the size of each part in a multipart upload (in bytes). If not set, defaults to 5MB (5242880 bytes). Minimum: 5MB (5242880 bytes), Maximum: 5GB (5368709120 bytes). | [optional] 
**parallel_transfers** | **int** | ParallelTransfers is the number of parallel transfers to use for S3 operations. If not set, defaults to 1. | [optional] 
**region** | **str** | Region contains the optional bucket region | [optional] 
**role_arn** | **str** | RoleARN is the Amazon Resource Name (ARN) of the role to assume. | [optional] 
**secret_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**session_token_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**use_sdk_creds** | **bool** | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


