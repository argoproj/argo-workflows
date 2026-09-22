

# IoArgoprojWorkflowV1alpha1S3Artifact

S3Artifact is the location of an S3 artifact

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**accessKeySecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**addressingStyle** | **String** | AddressingStyle defines how buckets are addressed by the S3 client. This is required for some S3-compatible providers that only support virtual-hosted-style bucket addressing.  Valid values are: - \&quot;\&quot; (default, auto-detect) - \&quot;path\&quot; - \&quot;virtual-hosted\&quot; |  [optional]
**bucket** | **String** | Bucket is the name of the bucket |  [optional]
**caSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**createBucketIfNotPresent** | [**IoArgoprojWorkflowV1alpha1CreateS3BucketOptions**](IoArgoprojWorkflowV1alpha1CreateS3BucketOptions.md) |  |  [optional]
**encryptionOptions** | [**IoArgoprojWorkflowV1alpha1S3EncryptionOptions**](IoArgoprojWorkflowV1alpha1S3EncryptionOptions.md) |  |  [optional]
**endpoint** | **String** | Endpoint is the hostname of the bucket endpoint |  [optional]
**insecure** | **Boolean** | Insecure will connect to the service with TLS |  [optional]
**key** | **String** | Key is the key in the bucket where the artifact resides |  [optional]
**region** | **String** | Region contains the optional bucket region |  [optional]
**roleARN** | **String** | RoleARN is the Amazon Resource Name (ARN) of the role to assume. |  [optional]
**secretKeySecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**sessionTokenSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**tokenExpirationInMinutes** | **Integer** | TokenExpirationInMinutes specifies the expiration time, in minutes, for the token obtained via AWS STS. It only applies to STS assumed-role credentials (when RoleARN is set) and STS web-identity credentials (when UseSDKCreds is set and web-identity environment variables are configured, e.g. EKS IRSA); it is forwarded as the STS DurationSeconds and must be between 15 and 720 minutes, matching the AWS STS allowed range, otherwise the request is rejected. It has no effect on static credentials or other SDK-resolved credential providers. |  [optional]
**useSDKCreds** | **Boolean** | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. |  [optional]



