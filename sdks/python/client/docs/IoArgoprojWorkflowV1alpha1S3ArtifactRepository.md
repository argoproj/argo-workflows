# IoArgoprojWorkflowV1alpha1S3ArtifactRepository

S3ArtifactRepository defines the controller configuration for an S3 artifact repository

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**bucket** | **str** | Bucket is the name of the bucket | [optional] 
**ca_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**create_bucket_if_not_present** | [**IoArgoprojWorkflowV1alpha1CreateS3BucketOptions**](IoArgoprojWorkflowV1alpha1CreateS3BucketOptions.md) |  | [optional] 
**enable_parallelism** | **bool** | EnableParallelism enables parallel upload/download for directories with many files or large files | [optional] 
**encryption_options** | [**IoArgoprojWorkflowV1alpha1S3EncryptionOptions**](IoArgoprojWorkflowV1alpha1S3EncryptionOptions.md) |  | [optional] 
**endpoint** | **str** | Endpoint is the hostname of the bucket endpoint | [optional] 
**file_count_threshold** | **int** | FileCountThreshold is the minimum number of files in a directory to trigger parallel operations. Default is 10. | [optional] 
**file_size_threshold** | **int** | FileSizeThreshold is the minimum file size in bytes to trigger multipart upload/download for single files. Default is 64MB. Files larger than this threshold will use multipart uploads with NumThreads parallelism. | [optional] 
**insecure** | **bool** | Insecure will connect to the service with TLS | [optional] 
**key_format** | **str** | KeyFormat defines the format of how to store keys and can reference workflow variables. | [optional] 
**key_prefix** | **str** | KeyPrefix is prefix used as part of the bucket key in which the controller will store artifacts. DEPRECATED. Use KeyFormat instead | [optional] 
**parallelism** | **int** | Parallelism is the number of concurrent workers for parallel operations. Default is 10. | [optional] 
**part_size** | **int** | PartSize is the part size in bytes for multipart uploads. Default is minio default, typically 128MB. Only used when FileSizeThreshold is exceeded. | [optional] 
**region** | **str** | Region contains the optional bucket region | [optional] 
**role_arn** | **str** | RoleARN is the Amazon Resource Name (ARN) of the role to assume. | [optional] 
**secret_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**session_token_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**use_sdk_creds** | **bool** | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


