# IoArgoprojWorkflowV1alpha1S3Artifact

S3Artifact is the location of an S3 artifact

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
**file_size_threshold** | **str** | FileSizeThreshold is the minimum file size to trigger multipart upload/download for single files. Default is 64MB. Files larger than this threshold will use multipart uploads with NumThreads parallelism. Can be specified as a Kubernetes resource quantity string (e.g., \&quot;64Mi\&quot;, \&quot;1Gi\&quot;). | [optional] 
**insecure** | **bool** | Insecure will connect to the service with TLS | [optional] 
**key** | **str** | Key is the key in the bucket where the artifact resides | [optional] 
**parallelism** | **int** | Parallelism is the number of concurrent workers for parallel operations. Default is 10. | [optional] 
**part_size** | **str** | PartSize is the part size for multipart uploads. Default is minio default, typically 128MB. Only used when FileSizeThreshold is exceeded. Can be specified as a Kubernetes resource quantity string (e.g., \&quot;128Mi\&quot;, \&quot;1Gi\&quot;). | [optional] 
**region** | **str** | Region contains the optional bucket region | [optional] 
**role_arn** | **str** | RoleARN is the Amazon Resource Name (ARN) of the role to assume. | [optional] 
**secret_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**session_token_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**use_sdk_creds** | **bool** | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


