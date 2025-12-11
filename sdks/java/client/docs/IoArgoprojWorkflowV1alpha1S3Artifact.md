

# IoArgoprojWorkflowV1alpha1S3Artifact

S3Artifact is the location of an S3 artifact

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**accessKeySecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**bucket** | **String** | Bucket is the name of the bucket |  [optional]
**caSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**createBucketIfNotPresent** | [**IoArgoprojWorkflowV1alpha1CreateS3BucketOptions**](IoArgoprojWorkflowV1alpha1CreateS3BucketOptions.md) |  |  [optional]
**enableParallelism** | **Boolean** | EnableParallelism enables parallel upload/download for directories with many files or large files |  [optional]
**encryptionOptions** | [**IoArgoprojWorkflowV1alpha1S3EncryptionOptions**](IoArgoprojWorkflowV1alpha1S3EncryptionOptions.md) |  |  [optional]
**endpoint** | **String** | Endpoint is the hostname of the bucket endpoint |  [optional]
**fileCountThreshold** | **Integer** | FileCountThreshold is the minimum number of files in a directory to trigger parallel operations. Default is 10. |  [optional]
**fileSizeThreshold** | **String** | FileSizeThreshold is the minimum file size to trigger multipart upload/download for single files. Default is 64MB. Files larger than this threshold will use multipart uploads with NumThreads parallelism. Can be specified as a Kubernetes resource quantity string (e.g., \&quot;64Mi\&quot;, \&quot;1Gi\&quot;). |  [optional]
**insecure** | **Boolean** | Insecure will connect to the service with TLS |  [optional]
**key** | **String** | Key is the key in the bucket where the artifact resides |  [optional]
**parallelism** | **Integer** | Parallelism is the number of concurrent workers for parallel operations. Default is 10. |  [optional]
**partSize** | **String** | PartSize is the part size for multipart uploads. Default is minio default, typically 128MB. Only used when FileSizeThreshold is exceeded. Can be specified as a Kubernetes resource quantity string (e.g., \&quot;128Mi\&quot;, \&quot;1Gi\&quot;). |  [optional]
**region** | **String** | Region contains the optional bucket region |  [optional]
**roleARN** | **String** | RoleARN is the Amazon Resource Name (ARN) of the role to assume. |  [optional]
**secretKeySecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**sessionTokenSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**useSDKCreds** | **Boolean** | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. |  [optional]



